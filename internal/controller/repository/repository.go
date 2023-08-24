/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repository

import (
	"context"
	"fmt"
	"log"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/MrVinkel/provider-bitbucketserver/apis/repository/v1alpha1"
	apisv1alpha1 "github.com/MrVinkel/provider-bitbucketserver/apis/v1alpha1"
	"github.com/MrVinkel/provider-bitbucketserver/internal/bitbucket"
	"github.com/MrVinkel/provider-bitbucketserver/internal/controller/features"
)

const (
	errNotRepository = "managed resource is not a Repository custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A BitbucketService provides operations against bitbucket
var (
	bitbucketService = func(baseURL string, creds []byte) (*bitbucket.BitBucketService, error) {
		client, err := bitbucket.NewClient(baseURL, string(creds))
		if err != nil {
			// crash if we get an error setting up client
			log.Fatalln(err)
		}
		service, err := bitbucket.NewService(client)
		if err != nil {
			// crash if we get an error setting up bitbucket service
			log.Fatalln(err)
		}
		return service, err
	}
)

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: bitbucketService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(baseURL string, creds []byte) (*bitbucket.BitBucketService, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(pc.Spec.BaseURL, data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API.
	service *bitbucket.BitBucketService
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	repoName := cr.Spec.ForProvider.Name
	projectName := cr.Spec.ForProvider.Project

	repository, err := c.service.Repositories.Get(ctx, &bitbucket.Repository{
		Name:    repoName,
		Project: projectName,
	})
	if err != nil {
		if errors.Is(err, bitbucket.ErrNotFound) {
			log.Printf("Repository (%s) does not exist in (%s)\n", repoName, projectName)
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "error fetching Bitbucket repository")
	}

	cr.SetConditions(xpv1.Available())
	cr.Status.AtProvider.ID = repository.ID

	// check description is up-to-date
	if repository.Description != cr.Spec.ForProvider.Description {
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	// check if groups are up-to-date
	groups, err := c.service.Repositories.GetGroups(ctx, repository)
	if !groupsEqual(cr.Spec.ForProvider.Groups, groups) {
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func groupsEqual(crGroups []v1alpha1.AdGroup, groups []bitbucket.Group) bool {
	if len(crGroups) != len(groups) {
		return false
	}

	for _, crGroup := range crGroups {
		found := false

		for _, group := range groups {
			if crGroup.Name == group.Name && crGroup.Permission == group.Permission {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}
	return true
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	cr.SetConditions(xpv1.Creating())

	repoToCreate := &bitbucket.Repository{
		Name:        cr.Spec.ForProvider.Name,
		Project:     cr.Spec.ForProvider.Project,
		Description: cr.Spec.ForProvider.Description,
	}

	log.Printf("Attempting to create Repository %+v\n", repoToCreate)

	repository, err := c.service.Repositories.Create(ctx, repoToCreate)
	if err != nil {
		log.Println(err)
		return managed.ExternalCreation{}, err
	}

	for _, g := range cr.Spec.ForProvider.Groups {
		group := bitbucket.Group{
			Name:       g.Name,
			Permission: g.Permission,
		}
		log.Printf("Creating permission %+v for repository %+v\n", group, repository)
		err = c.service.Repositories.AddGroup(ctx, repository, &group)
		if err != nil {
			log.Printf("Error creating permission: %v", err)
			return managed.ExternalCreation{}, err
		}
	}
	log.Printf("Finished creating repository %+v\n", repository)

	meta.SetExternalName(cr, fmt.Sprint(repository.Name))

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	log.Printf("Attempting to update repository %s\n", cr.Name)

	repoToUpdate := &bitbucket.Repository{
		Name:        cr.Spec.ForProvider.Name,
		Project:     cr.Spec.ForProvider.Project,
		Description: cr.Spec.ForProvider.Description,
	}

	repo, err := c.service.Repositories.Update(ctx, repoToUpdate)
	if err != nil {
		log.Println(err)
		return managed.ExternalUpdate{}, err
	}

	groups, err := c.service.Repositories.GetGroups(ctx, repo)
	if err != nil {
		log.Println(err)
		return managed.ExternalUpdate{}, err
	}

	// Update all groups
	for _, group := range cr.Spec.ForProvider.Groups {
		log.Printf("Updating permission %+v for repository %+v\n", group, repo)
		err = c.service.Repositories.AddGroup(ctx, repo, &bitbucket.Group{Name: group.Name, Permission: group.Permission})
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	// Delete unknown groups
	for _, group := range groups {
		found := false
		for _, crGroup := range cr.Spec.ForProvider.Groups {
			if group.Name == crGroup.Name {
				found = true
				break
			}
		}
		if !found {
			err = c.service.Repositories.RevokeGroup(ctx, repo, &group)
			if err != nil {
				return managed.ExternalUpdate{}, err
			}
		}
	}

	log.Printf("Finished updating repository %+v\n", repo)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return errors.New(errNotRepository)
	}

	log.Printf("Attempting to delete repository %s\n", cr.Spec.ForProvider.Name)

	cr.SetConditions(xpv1.Deleting())

	return c.service.Repositories.Delete(ctx, &bitbucket.Repository{
		Name:    cr.Spec.ForProvider.Name,
		Project: cr.Spec.ForProvider.Project,
	})
}
