

# OC Diff Tool

## Goal

Allow developers to work with version controlled configuration files. All resources in Openshift should be created and modified based on these configuration files.

## Benefits

* New projects etc. can be created based on configuration.
* If resources vanish in OC, they can be recreated from config.
* Solutions / patterns can be easily copied between projects.
* Rollback is easy.
* Cloning becomes unecessary.

## Usage

There are three main commands: `export`, `status` and `update`.

`export` allows you to export configuration found in an OC project to YAML templates, which are cleaned up to be used with `ocdiff`. By default, one template is written to STDOUT, but if you pass `--write-files-by-kind` then one YAML file is written per resource type.

`status` shows you the drift between the current configuration in the OC project and the desired configuration in the YAML templates (which are either in the current directory or in `--template-dir`). There are three main aspects to this:
1. By default, all resource types are compared, but you can limit to specific ones e.g. with `status pvc,dc`.
2. The desired configuration is computed by processing the local YAML templates. It is possible to pass `--labels`, `--param` and `--param-file` to the `status` command to influence the generated config. Those 3 flags are passed as-is to the underlying `oc process` command. As `ocdiff` allows you to work with multiple templates, there is an additional `--param-dir` flag, which you can use to point to a folder containing param files for each template (they are connected by naming convention, so for template `foo.yml` the corresponding param file would be `foo.env`). When passing `--param` or `--param-file`, you might encounter that not all of your templates declare the same parameters, which leads to Openshift aborting the operation. Use `--ignore-unknown-parameters` to prevent this.
3. In order to calculate drift correctly, the whole OC project is compared against your configuration. If you want to compare a subset only (e.g. all resources related to one microservice), it is possible to narrow the scope by passing `--selector`. Further, you can specify individual resources, e.g. `dc/foo,bc/bar`. If for some reason you do not have all resources described in your local configuration, but want to prevent deletion of resources in Openshift, use `--upsert-only`.

Finally, `update` will compare current vs. desired configuration exactly like `status` does, but if any drift is detected, it asks to update the OC project with your desired state. A subsequent run of either `status` or `update` should show no drift. To help usage inside of scripts, the confirmation can be disabled with `--non-interactive`.

All commands depend on a current OC session and accept a `--namespace` (if none is given, the current one is used). To help with debugging (e.g. to see the commands which are executed in the background), use `--verbose`.

All options can be inspected with `ocdiff help`.

## Working with Secrets

Keeping the Openshift configuration under version control necessitates to store secrets. To this end `ocdiff` comes with a `secrets` subcommand that allows to encrypt those secrets. The subcommands offers to `edit`, `re-encrypt` and `reveal` secrets, as well as adding new keys via `generate-key`.

`secrets edit foo.env` allows you to encrypt individual parameters with PGP by adding `.ENC` to the param name, e.g. `PASSWORD.ENC=c2VjcmV0`. Notice the value is base64-encoded as Openshift stores secrets base64-encoded. You can also enter plain text by using `PASSWORD.STRING=secret` which will get transformed to `PASSWORD.ENC=c2VjcmV0` automatically. When saved, the param value will be encrypted for all public keys in `--public-key-dir` (defaulting to the current directory). To read a file with encrypted params (e.g. to edit the secrets or compare the status between config and current state), you need your private key available at `--private-key` (defaulting to `private.key`).

When a public key is added or removed, it is required to `secrets re-encrypt` all secrets. This decrypts all `*.env` files and writes them again using the provided public keys.

The `secrets reveal` command shows the param file after decrypting and decoding the values so that you can see the clear text secrets.

Finally, to ease PGP key management, `secrets generate-key john.doe@domain.com` generates a PGP keypair, writing the public key to `john-doe.key` (which should be committed) and the private key to `private.key` (which MUST NOT be committed).

## Advanced Usage

### Command Completion

BASH/ZSH completion is available. Add this into `.bash_profile` or equivalent:
```
eval "$(ocdiff --completion-script-$(echo $SHELL | awk -F/ '{print $NF}'))"
```

## Background

### Problem

Kubernetes and Openshift insert additional properties and modify existing ones automatically. Configuration files under version control cannot simply replace the "live" ones, but need to be merge in. However, the `export` command OC already strips the config from most unwanted things.

### Context / Alternatives

* kubediff - for kubernetes, written in python
* `oc apply` - it seems to be able to do the update, but there is no diffing and no deletion
* Stiching together a couple of commands (export, template processing, etc.) is cumbersome






