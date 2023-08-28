# Crossplane provider for BitBucket server 

This project contains a crossplane provider for managing Bitbucket Server resources

Manages: 

- Projects
- Repositories
- Group permissions on repositories

## How to

### Add provider resource type

1. Run the addtype target:

    `make provider.addtype provider=<Pascal case> group=<api group> kind=<type of resource>`

    Example:

    `make provider.addtype provider=BitbucketServer group=repository kind=Repository`
2. This will create 3 files under `apis/<group>/v1aplha1`. Add fields to `<kind>Parameters` struct in the `<group>_types.go` file.
3. Generate the yaml schemas by running: `make generate`
4. Add the new api SchemaBuilder to `apis/<provider>.go` in the init function
5. Implement the observe, create, update and delete methods in the generated `internal/controller/<kind>.go`
6. Register the new controller in `internal/controller/<provider>.go`

### Test the provider in kind

1. Run `make dev` 
   
   This will start a k8s cluster and start the provider (you can ctrl+c the provider and rerun it with `make run` without having to restart the cluster)
2. Edit or add the yaml files under `examples` to configure the provider and to make example resources.
3. Apply the yaml files with `kubectl apply -f examples/<yaml file>` and see how the provider responds in the log.

### Release

1. Create a tag `git tag v1.2.3` and push it `git push origin v1.2.3`
2. Run the CI pipeline manual on github to promote it to docker hub
