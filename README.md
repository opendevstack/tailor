# Tailor

Tailor is a tool to drive the configuration of your OpenShift resources via
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

The latest release is 1.1.4 and requires oc >= v3.9.0. OpenShift 4 is not officially supported yet although 1.0.0 and above work in principle.
Please have a look at the [changelog](https://github.com/opendevstack/tailor/blob/master/CHANGELOG.md) when upgrading.

MacOS:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.1.4/tailor-darwin-amd64" && \
chmod +x tailor-darwin-amd64 && mv tailor-darwin-amd64 /usr/local/bin/tailor
```

Linux:

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.1.4/tailor-linux-amd64" && \
chmod +x tailor-linux-amd64 && mv tailor-linux-amd64 /usr/local/bin/tailor
```

Windows (using Git Bash):

```
curl -LO "https://github.com/opendevstack/tailor/releases/download/v1.1.4/tailor-windows-amd64.exe" && \
chmod +x tailor-windows-amd64.exe && mv tailor-windows-amd64.exe /mingw64/bin/tailor.exe
```

## Usage

There are three main commands: `diff`, `apply` and `export`. All commands depend on a current OpenShift session. To help with debugging (e.g. to see the `oc` commands which are executed in the background), use `--verbose`. More commands and options can be discovered via `tailor help`. All options can also be read from a file to ease usage, see section [Tailorfile](#tailorfile).

### `tailor diff / apply`
`diff` compares the current state of a namespace with its desired state, and shows the resulting drift. The current state is determined by exporting the resources in the OpenShift cluster (via `oc get --export`). The desired state is computed by processing local OpenShift templates (via `oc process`), using any parameters given via the CLI or param files.

`apply` does exactly the same as `diff`, but then asks whether to reconcile the drift, which can be done as a whole or on a per-resource basis.

There are many options to control how the comparison is performed:

* The namespace which is compared can be specificed via `--namespace|-n`. If not given, it defaults to the active namespace of the session.
* Templates (`*.yml` files) are taken from `--template-dir|-t` (defaulting to the working dir).
* Param files (`*.env` files) are taken from `--param-dir|-p` (defaulting to a directory with the same name as the target namespace in the current working dir; otherwise the working dir itself). Each param file is then used when processing the "corresponding" template (e.g. `foo.env` for template `foo.yml`).
* Param files can also be referenced via `--param-file`. If a file named `<namespace>.env` exists in the working dir, it is automatically passed as `--param-file`.
* Parameters can also be specified directly via `--param FOO=bar`.
* If at least one of the processed templates does not consume all given parameters, `oc process` will fail to highlight this problem. To squelch this message, use `--ignore-unknown-parameters`.
* By default, all resources in the namespace are compared, but you can adjust this by:
  * adding specific types as arguments to the command, e.g. `tailor diff pvc,dc`
  * passing `--selector/-l`, e.g. `-l app=foo` (multiple labels are comma-separated, and need to apply all)
  * specifying an individual resource, e.g. `dc/foo`
  * excluding resources via `--exclude|-e` (targeting types, resources or labels; e.g. `-e bc`, `-e dc/foo`, `-e app=foo`)
* Sometimes there is state in the OpenShift cluster which is difficult to "know" in the templates. Tailor allows to keep the state of a field in OpenShift via `--preserve` (e.g. `--preserve bc`, `--preserve bc:foobar`, `--preserve bc:/spec/output/to/name`).
* Changing the value of some fields (such as the `host` of a `Route`) is not allowed in OpenShift. Tailor detects if you do so and displays a warning that it would need to recreate the resource to apply the change. You may then permit this via `--allow-recreate` or avoid drift on such fields via `--preserve-immutable-fields`.
* Drift on `Secret` resources is hidden by default for security reasons, and may be shown by passing `--reveal-secrets`.

### `tailor export`
Export configuration of resources found in an OpenShift namespace to a cleaned
YAML template, which is written to `STDOUT`. Tailor applies three optimisations to the result:

- All fields controlled by the cluster are removed, such as `/metadata/creationTimestamp`.
- Unless `--with-annotations` is given, some annotations (`kubectl.kubernetes.io/last-applied-configuration`, `openshift.io/image.dockerRepositoryCheck`) are removed. It is possible to remove further annotation(s) via `--trim-annotation`, either by exact match or by prefix match (e.g. `openshift.io/`).
- Hardcoded occurences of the namespace are replaced with an automatically supplied parameter `TAILOR_NAMESPACE` so that the exported template can be used against multiple OpenShift projects (can be disabled by passing `--with-hardcoded-namespace`).


## How-To

### Template Authoring

Please consult the [OpenShift Templates documentation](https://docs.openshift.com/container-platform/3.11/dev_guide/templates.html) on how to write templates to express the desired state. Tailor processes the templates using standard `oc process`, before the resulting resource list gets applied via `oc apply`. For in-depth knowledge about how the configuration in the templates gets applied to the current state in the cluster, read [Declarative Management of Kubernetes Objects Using Configuration Files](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/).

Following is some guidance on how to author templates:

* If the template specifies a parameter `TAILOR_NAMESPACE`, it is automatically filled based on the namespace against which Tailor is executed.
* Some resource fields have useful server defaults (such as `.spec.host` of `Route` resources or `.spec.storageClassName` of `PersistentVolumeClaim` resources). It is possible to leave them out of the template, but Tailor will detect drift after the resource has been created (because the value is present in the live configuration, but absent in the template). One can use e.g. `--preserve route:/spec/host` to prevent this. Alternatively, some of those fields are also immutable, so using `--preserve-immutable-fields` can also work well.
* Often it is easier to start authoring templates by exporting live configuration instead of starting from scratch. Also, sometimes it can be easier to apply a change in the UI and then figure out what needs to be updated in the template by running `tailor diff`.

### Working with Secrets

Keeping the OpenShift configuration under version control necessitates to store secrets. To make it easy to do so in a safe fashion, Tailor comes with a `secrets` subcommand that allows to encrypt those secrets using PGP. The subcommands offers to `edit`, `re-encrypt` and `reveal` secrets, as well as adding new keypairs via `generate-key`.

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

## FAQ / Troubleshooting

### Tailor does not recognize a certain resource kind

Tailor currently supports `BuildConfig`, `CronJob`, `Job`, `Deployment`, `DeploymentConfig`, `ImageStream`, `LimitRange`, `PersistentVolumeClaim`, `ResourceQuota`, `RoleBinding`, `Route`, `Secret`, `Service`, `ServiceAccount`, `Template`. Some resources like `Build`, `Event`, `ImageStreamImage`, `ImageStreamTag`, `PersistentVolume`, `Pod`, `ReplicationController` are not supported by design as they are created and managed automatically by OpenShift. If you want to control a resource with Tailor that is not supported yet, but would be suitable, please [open an issue](https://github.com/opendevstack/tailor/issues/new).

### Why is it required to specify fields which have server defaults?

When a field (such as `.spec.revisionHistoryLimit` of `DeploymentConfig` resources) is absent from a template, the server will default it when the template is applied. However, in subsequent runs, Tailor will detect drift for that field, suggesting to remove it (even though the path would not actually be removed as the server would default it again). Technically, it would be possible to prevent detecting drift for those circumstances (based on previously applied configuration), but it would run the risk that changes are made in the UI which would not be detected by Tailor, and therefore lead to situations where the live configuration does not match the desired state. Because of this, Tailor detects drift unless the field is also defined in the template, or the live value is preserved (via `--preserve ...`)

---

![](https://github.com/opendevstack/tailor/workflows/Continous%20Integration%20Tests/badge.svg?branch=master)




