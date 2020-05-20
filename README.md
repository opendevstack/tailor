# Tailor

Tailor is a tool to drive the configuration of your OpenShift cluster via
templates under version control. Any drift between your desired state (the YAML templates) and the current state (resources in the cluster) can be detected, reviewed and reconciled using a CLI interface. Tailor is required by other OpenDevStack repositories, but is fully standalone and may be used in completely different contexts. It uses `oc` commands under the hood, and is based on
plain OpenShift templates.

## Benefits of using infrastructure-as-code

* Applications consisting of multiple resources or even whole namespaces can be created / cloned in one go based on configuration.
* If resources are (accidentally) removed from the OpenShift cluster, they can be recreated from config.
* Rollback of changes in the cluster is easy as configuration is under version control.
* Visibility into which changes were made and by whom.

## Benefits of using Tailor over plain `oc` commands

* Display and review drift before applying the desired state.
* Support for encrypted secrets to avoid storing credentials in clear text in your repository.
* Preserving the values of certain paths in "live" resources (e.g. values of immutable fields or any other field controlled by other means).
* Simple interface which avoids having to stitch multiple commands together and massage their output to achieve the desired result.
* Options can be stored in a `Tailorfile` to ease invocation and ensure consistency within teams.

If you are interested how Tailor compares to other tools such as Helm, please see [comparison to other tools](https://github.com/opendevstack/tailor#comparison-to-other-tools).

## Installation

The latest release is 1.0.0 and requires oc >= v3.9.0. OpenShift 4 is not officially supported yet although 1.0.0 and above work in principle.
Please have a look at the [changelog](https://github.com/opendevstack/tailor/blob/master/CHANGELOG.md) when upgrading.

MacOS:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.0.0/tailor-darwin-amd64" && \
chmod +x tailor-darwin-amd64 && mv tailor-darwin-amd64 /usr/local/bin/tailor
```

Linux:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.0.0/tailor-linux-amd64" && \
chmod +x tailor-linux-amd64 && mv tailor-linux-amd64 /usr/local/bin/tailor
```

Windows (using Git Bash):

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.0.0/tailor-windows-amd64.exe" && \
chmod +x tailor-windows-amd64.exe && mv tailor-windows-amd64.exe /mingw64/bin/tailor.exe
```

## Usage

There are three main commands: `diff`, `apply` and `export`.



### `diff`
Show drift between the current state in the OpenShift cluster and the desired
state in the YAML templates. There are three main aspects to this:
1. By default, all resource types are compared, but you can limit to specific ones, e.g. `diff pvc,dc`.
2. The desired state is computed by processing the local YAML templates. It is possible to pass `--labels`, `--param` and `--param-file` to the `diff` command to influence the generated config. Those 3 flags are passed as-is to the underlying `oc process` command. If a file named `<namespace>.env` exists in the working directory, it is automatically passed as `--param-file`. Additionally, as Tailor allows you to work with multiple templates, there is a `--param-dir="<namespace>|."` flag, which you can use to point to a folder containing param files corresponding to each template (e.g. `foo.env` for template `foo.yml`).
3. In order to calculate drift correctly, the whole OpenShift namespace is compared against your configuration. If you want to compare a subset only (e.g. all resources related to one microservice), it is possible to narrow the scope by passing `--selector/-l`, e.g. `-l app=foo` (multiple labels are comma-separated, and need to apply all). Further, you can specify an individual resource, e.g. `dc/foo`.

### `apply`
This command will compare current vs. desired state exactly like `diff` does,
but if any drift is detected, it asks to apply the OpenShift namespace with your desired state. A subsequent run of either `diff` or `apply` should show no drift.

### `export`
Export configuration of resources found in an OpenShift namespace to a cleaned
YAML template, which is written to `STDOUT`. Tailor applies three optimisations to the result:

- All fields controlled by the cluster are removed, such as `/metadata/creationTimestamp`.
- Unless `--with-annotations` is given, annotations are removed.
- Hardcoded occurences of the namespace are replaced with an automatically supplied parameter `TAILOR_NAMESPACE` so that the exported template can be used against multiple OpenShift projects (can be disabled by passing `--with-hardcoded-namespace`).

### General Usage Notes
All commands depend on a current OpenShift session and accept a `--namespace` flag (if none is given, the current one is used). To help with debugging (e.g. to see the `oc` commands which are executed in the background), use `--verbose`. More options can be displayed with `tailor help`.


## How-To

### Template Authoring

Please consult the [OpenShift Templates documentation](https://docs.openshift.com/container-platform/3.11/dev_guide/templates.html) on how to write templates to express the desired state. Tailor processes the templates using standard `oc process`, with one addition: if the template specifies a parameter `TAILOR_NAMESPACE`, it is automatically filled based on the namespace against which Tailor is executed.

For in-depth knowledge about how the configuration in the templates gets applied to the current state in the cluster, read [Declarative Management of Kubernetes Objects Using Configuration Files](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/).

### Working with Secrets

Keeping the OpenShift configuration under version control necessitates to store secrets. To this end Tailor comes with a `secrets` subcommand that allows to encrypt those secrets using PGP. The subcommands offers to `edit`, `re-encrypt` and `reveal` secrets, as well as adding new keypairs via `generate-key`.

In general, secrets are just a special kind of params. Typically, params are located in `*.env` files, e.g. `FOO=bar`. Secrets an be kept in a `*.env.enc` file, where each line is e.g. `QUX=<encrypted content>`. When Tailor is processing templates, it merges `*.env` and `*.env.enc` files together. All params in `.env.enc` files are base64-encoded automatically by Tailor so that they can be used directly in OpenShift `Secret` resources. If you have a secret value that is a multiline string (such as a certificate), you can base64-encode it (e.g. `cat cert | base64`) and add the encoded string as a parameter into the `.env.enc` file like this: `FOO.B64=abc...`. The `.B64` suffix tells Tailor that the value is already in base64 encoding.

In order to create and edit `*.env.enc` files, Tailor offers an `edit` command. `secrets edit foo.env.enc` opens a terminal editor, in which you can enter the params in plain, e.g. `PASSWORD=s3cr3t`. When saved, every param value will be encrypted for all public keys in `--public-key-dir="public-keys|."`. To read a file with encrypted params (e.g. to edit the secrets or compare the diff between desired and current state), you need your private key available at `--private-key="private.key"`.

When a public key is added or removed, it is required to run `secrets re-encrypt`.
This decrypts all params in `*.env.enc` files and writes them again using the provided public keys.

The `secrets reveal foo.env.enc` command shows the param file after decrypting
the param values so that you can see the clear text secrets.

Finally, to ease PGP management, `secrets generate-key john.doe@domain.com` generates a PGP keypair, writing the public key to `john-doe.key` (which should be committed) and the private key to `private.key` (which MUST NOT be committed).


### Permissions

Tailor needs access to a resource in order to be able to compare it. This means that to properly compare all resources, the user of the OpenShift session that Tailor makes use of needs to have enough rights. Failing, Tailor will error
as it cannot compare some resources. To prevent this from happening, exclude the resource types (e.g. `rolebinding` and `serviceaccount`) that you do not have access to.


### Tailorfile

Instead of passing flags ad-hoc to `tailor`, all options can be specified in a `Tailorfile`, which is a simple line-delimited file, e.g.:
```
template-dir foo
param-dir bar

param FOO=bar
param BAZ=qux

bc,is,dc,svc
```
Please note that boolean flags need to be specified with a value, e.g. `upsert-only true`.

Tailor will automatically pick up any file named `Tailorfile.<namespace>` or `Tailorfile` in the working directory. Alternatively, a specific file can be selected via `tailor -f somefile`.

### Command Completion

BASH/ZSH completion is available. Add this into `.bash_profile` or equivalent:
```
eval "$(tailor --completion-script-$(echo $SHELL | awk -F/ '{print $NF}'))"
```

## Comparison to other tools

### Tailor vs. Helm
* Compared to Helm v2, Tailor does not need a highly privileged Tiller. Helm v3 does not need it either.
* Tailor works with plain OpenShift templates instead of Helm charts. OpenShift templates are simpler, and can easily be generated from existing resources in OpenShift.
* Tailor has a narrower scope - it is basically an "`oc` on steroids". Helm has more extensive features like searching for charts etc.
* Tailor targets OpenShift, Helm targets Kubernetes. Using Helm for OpenShift has limitations / bugs around dealing with OpenShift resources such as `BuildConfig` or `Route`.
* Tailor allows to check for drift, and allows to review the difference between live configuration and desired state before applying.

## Troubleshooting

### Tailor does not recognize a certain resource kind

Tailor currently supports `BuildConfig`, `CronJob`, `Deployment`, `DeploymentConfig`, `ImageStream`, `LimitRange`, `PersistentVolumeClaim`, `ResourceQuota`, `RoleBinding`, `Route`, `Secret`, `Service`, `ServiceAccount`, `Template`. Some resources like `Build`, `Event`, `ImageStreamImage`, `ImageStreamTag`, `PersistentVolume`, `Pod`, `ReplicationController` are not supported by design as they are created and managed automatically by OpenShift. If you want to control a resource with Tailor that is not supported yet, but would be suitable, please [open an issue](https://github.com/opendevstack/tailor/issues/new).

---

[![Build Status](https://travis-ci.com/opendevstack/tailor.svg?branch=master)](https://travis-ci.com/opendevstack/tailor)




