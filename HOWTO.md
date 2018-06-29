# How-To

## Working with Images

When a template references an image, e.g. in a deployment configuration, it can be tricky to keep this in sync with Openshift. For example, if the template uses `registry.domain.com/foo/bar@sha256:a1b2c3`, then `ocdiff` reports drift when a new build is triggered in Openshift since Openshift overwrites the SHA in the deployment config.

To prevent this from happening, you can specify a tag in the template instead, e.g. `registry.domain.com/foo/bar:latest`. While Openshift still resolves this tag upon deployment to an actual SHA, `ocdiff` stores the original value from the template within Openshift so that the current and desired state are in sync. One (minor) downside of this approach is that if an image reference is updated in the UI while the annotation inserted by `ocdiff` is not, `ocdiff` will not detect this change properly.

Another complication arises when provisioning a deployment config referencing a non-existant image stream. This can happen e.g. if you "clone" a set of resources into a different OC namespace. The deployment config will fail since it cannot find the image, and you need to manually trigger a build. Currently `ocdiff` does not offer a solution for this as it is not clear (yet) what the right way to handle this is.

## Permissions

`ocdiff` needs access to the resources to be able to compare state. This means that to properly compare all resources, the user of the OC session that `ocdiff` makes use of needs to be admin. If you are not admin, `ocdiff` will fail. To prevent this from happening, exclude the resource types (e.g. rolebindings and serviceaccounts) that you do not have access to.
