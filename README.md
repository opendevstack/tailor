

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

`export` allows you to export configuration found in an OC project to YAML templates. Those templates are cleaned automatically.

`status` shows you the drift between the current configuration in the OC project and the desired configuration in the YAML templates. There are three main aspects to this:
1. By default, all resource types are compared, but you can limit to specific ones e.g. with `status pvc,dc`.
2. The desired configuration is computed by processing the local YAML templates. It is possible to pass `--labels`, `--param` and `--param-file` to the `status` command to influence the generated config. Those 3 flags are passed as-is to the underlying `oc process` command. As `ocdiff` allows you to work with multiple templates, there is an additional `--param-dir` flag, which you can use to point to a folde containing param files for each template (they are connected by naming convention, so for template `foo.yml` the corresponding param file would be `foo.env`).
3. In order to calculate drift correctly, the whole OC project is compared against your configuration. If you want to compare a subset only (e.g. all resources related to one microservice), it is possible to narrow the scope by passing `--selector`.

Finally, `update` will compare current vs. desired configuration exactly like `status` does, but if any drift is detected, it asks to update the OC project with your desired state. A subsequent run of either `status` or `update` should show no drift. To help usage inside of scripts, the confirmation can be disabled with `--non-interactive`.

All commands depend on a current OC session and accept a `--namespace` (if none is given, the current one is used). To help with debugging (e.g. to see the commands which are executed in the background), use `--verbose`.

All options can be inspected with `ocdiff help`.

## Background

### Problem

Kubernetes and Openshift insert additional properties and modify existing ones automatically. Configuration files under version control cannot simply replace the "live" ones, but need to be merge in. However, the `export` command OC already strips the config from most unwanted things.

### Context / Alternatives

* kubediff - for kubernetes, written in python
* `oc apply` - it seems to be able to do the update, but there is no diffing and no deletion
* Stiching together a couple of commands (export, template processing, etc.) is cumbersome






