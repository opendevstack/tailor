# Tailor

Tailor is a layer on top of the `oc` CLI command that allows you to keep your OpenShift templates under version control and keep the cluster in sync with them. Any drift between your desired state (YAML templates) and the current state (resources in the cluster) can be detected, reviewed and reconciled with Tailor. The tool is required by other OpenDevStack repositories, but is fully standalone and may be used in completely different contexts.

## Benefits

* Applications consisting of multiple resources or even whole namespaces can be created / cloned in one go based on configuration.
* If resources are (accidentally) removed from the OpenShift cluster, they can be recreated from config.
* Rollback of changes in the cluster is easy as configuration is under version control.
* Visibility of what changes were made and by whom.
* Problems that arise from the fact that OpenShift combines configuration and state into one object (even in the export) are addressed by `tailor`. E.g. annotation injected into resources or modification of image references etc. are dealt with.
* Support for encrypted secrets to avoid storing credentials in clear text in your repository.

## Installation

The latest release is 0.7.0 and requires oc >= v3.9.0.
Please have a look at the [changelog](https://github.com/opendevstack/tailor/blob/master/CHANGELOG.md) when upgrading.

MacOS:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v0.7.0/tailor_darwin_amd64" && \
mv tailor_darwin_amd64 /usr/local/bin/tailor
```

Linux:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v0.7.0/tailor_linux_amd64" && \
mv tailor_linux_amd64 /usr/local/bin/tailor
```

Windows (using Git Bash):

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v0.7.0/tailor_windows_amd64.exe" && \
mv tailor_windows_amd64.exe /mingw64/bin/tailor.exe
```

## Usage

There are three main commands: `export`, `status` and `update`.

`export` allows you to export configuration found in an OpenShift namespace to a cleaned YAML template, which is written to STDOUT.

`status` shows you the drift between the current state in the OpenShift namespace and the desired state in the YAML templates (located in `--template-dir="."`). There are three main aspects to this:
1. By default, all resource types are compared, but you can limit to specific ones, e.g. `status pvc,dc`.
2. The desired state is computed by processing the local YAML templates. It is possible to pass `--labels`, `--param` and `--param-file` to the `status` command to influence the generated config. Those 3 flags are passed as-is to the underlying `oc process` command. As `tailor` allows you to work with multiple templates, there is an additional `--param-dir="<namespace>|."` flag, which you can use to point to a folder containing param files corresponding to each template (e.g. `foo.env` for template `foo.yml`).
3. In order to calculate drift correctly, the whole OpenShift namespace is compared against your configuration. If you want to compare a subset only (e.g. all resources related to one microservice), it is possible to narrow the scope by passing `--selector/-l`, e.g. `-l app=foo`. Further, you can specify anindividual resource, e.g. `dc/foo`.

Finally, `update` will compare current vs. desired state exactly like `status` does, but if any drift is detected, it asks to update the OpenShift namespace with your desired state. A subsequent run of either `status` or `update` should show no drift.

All commands depend on a current OpenShift session and accept a `--namespace` flag (if none is given, the current one is used). To help with debugging (e.g. to see the commands which are executed in the background), use `--verbose`. More options can be displayed with `tailor help`.

## How-To

### Working with Secrets

Keeping the OpenShift configuration under version control necessitates to store secrets. To this end `tailor` comes with a `secrets` subcommand that allows to encrypt those secrets using PGP. The subcommands offers to `edit`, `re-encrypt` and `reveal` secrets, as well as adding new keypairs via `generate-key`.

`secrets edit foo.env` opens a terminal editor, allowing you to encrypt individual parameters by adding `.ENC` to the param name, e.g. `PASSWORD.ENC=c2VjcmV0`. Notice that the value is base64-encoded as OpenShift stores secrets base64-encoded. You can also enter plain text by using `PASSWORD.STRING=secret` which will get transformed to `PASSWORD.ENC=c2VjcmV0` automatically. When saved, every `.ENC` param value will be encrypted for all public keys in `--public-key-dir="public-keys|."`. To read a file with encrypted params (e.g. to edit the secrets or compare the status between desired and current state), you need your private key available at `--private-key="private.key"`.

When a public key is added or removed, it is required to run `secrets re-encrypt`. This decrypts all params in `*.env` files and writes them again using the provided public keys.

The `secrets reveal` command shows the param file after decrypting and decoding the param values so that you can see the clear text secrets.

Finally, to ease PGP management, `secrets generate-key john.doe@domain.com` generates a PGP keypair, writing the public key to `john-doe.key` (which should be committed) and the private key to `private.key` (which MUST NOT be committed).

### Working with Images

When templates reference images (e.g. in a DeploymentConfig) it can be tricky to keep them in sync with OpenShift, as OpenShift resolves the image reference (e.g. `foo:latest`) to a specific version (e.g. `foo@sha256:a1b2c3`). Consequently, the current and desired state are out of sync. A similar problem is that new builds will produce images in the image stream unknown at the time when the local template is authored.

To prevent `tailor` from constantly reporting drift and "resetting" the current state to the local template, `tailor` stores the original value from the template within OpenShift (in an annotation) so that the current and desired state can be compared properly. Keep in mind that you should specify an image tag in the template, e.g. `registry.domain.com/foo/bar:latest`. Images located in the OCP registry can be referenced with `foo/bar:latest`. One (minor) downside of the `tailor` approach is that if an image reference is updated in the OpenShift UI while the annotation inserted by `tailor` is not, `tailor` will not detect this change properly. Therefore, always treat the local configuration as the single source of truth.

Another complication arises when provisioning a DeploymentConfig referencing a non-existant image stream. This can happen e.g. if you "clone" a set of resources into a different OpenShift namespace. The DeploymentConfig will not deploy since it cannot find the image, and you need to manually trigger a build. Currently `tailor` does not offer a solution for this as it is not clear (yet) what the right way to "solve" this is.

### Permissions

`tailor` needs access to a resource in order to be able to compare it. This means that to properly compare all resources, the user of the OpenShift session that `tailor` makes use of needs to be admin. If you are not admin, `tailor` will fail as it cannot compare some resources. To prevent this from happening, exclude the resource types (e.g. `rolebinding` and `serviceaccount`) that you do not have access to.

### Tailorfile

Since specifying all params correctly can be daunting, and it isn't easy to share how `tailor` should be invoked, `tailor` supports setting flags via a `Tailorfile`. This is simply a line-delimited file, e.g.:
```
template-dir foo
param-dir bar

param FOO=bar
param BAZ=qux

bc,is,dc,svc
```

### Command Completion

BASH/ZSH completion is available. Add this into `.bash_profile` or equivalent:
```
eval "$(tailor --completion-script-$(echo $SHELL | awk -F/ '{print $NF}'))"
```

---

[![Build Status](https://travis-ci.com/opendevstack/tailor.svg?branch=master)](https://travis-ci.com/opendevstack/tailor)




