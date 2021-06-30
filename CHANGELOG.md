# Changelog

## Unreleased

## [1.3.2] - 2021-06-30

### Fixed

- Avoid panic on template with empty parameters([#238](https://github.com/opendevstack/tailor/pull/238))
- Preserve svc cluster ips by default ([#242](https://github.com/opendevstack/tailor/pull/242))

## [1.3.1] - 2021-02-11

### Added

- Add support for `StatefulSet` resources ([#235](https://github.com/opendevstack/tailor/pull/235)).

## [1.3.0] - 2020-11-13

### Changed

- Use `oc get` without `--export` flag, which has been removed in Kubernetes 1.19 ([#230](https://github.com/opendevstack/tailor/pull/230)).

## [1.2.2] - 2020-10-27

### Added

- Handle `HorizontalPodAutoscaler` resoures ([#227](https://github.com/opendevstack/tailor/pull/227)).

## [1.2.1] - 2020-09-21

### Changed

- Export resources and process templates concurrently ([#220](https://github.com/opendevstack/tailor/pull/220)). This should not cause any change in behaviour, but you should notice Tailor being faster, especially against slow clusters.

### Fixed

- Ignore `managedFields` ([#222](https://github.com/opendevstack/tailor/pull/222)). The fixed problem only occurs when Tailor is used against OpenShift 4.

## [1.2.0] - 2020-08-10

### Added

- Handle `Job` resoures ([#212](https://github.com/opendevstack/tailor/pull/212)).

### Changed

- Do not print timestamps when errors occur ([b378a89](https://github.com/opendevstack/tailor/commit/b378a89be243037c00bee0d2c9fdb9870b69b69e)).
- Change test infrastructure to GitHub Actions ([#186](https://github.com/opendevstack/tailor/pull/186) and [#216](https://github.com/opendevstack/tailor/pull/216)), and implement easier to maintain approach for end-to-end tests ([#215](https://github.com/opendevstack/tailor/pull/215) and [#219](https://github.com/opendevstack/tailor/pull/219)).

### Fixed

- Export `LimitRange` and `Quota` resources ([decfae9](https://github.com/opendevstack/tailor/commit/decfae93d5542c0d9ca9e9b3c61e8e697d88c124)). Support for those resources was added in 1.0.0, but they were not exported, only applied. Note that this change may lead to differences in diff output compared to 1.1.4.

## [1.1.4] - 2020-07-20

### Fixed

- Replace only whole words with `${TAILOR_NAMESPACE}` ([#211](https://github.com/opendevstack/tailor/pull/211)). Note that while this should yield better results than the previous behaviour, there might be rare situations where Tailor misses to replace the namespace now.

## [1.1.3] - 2020-07-01

### Added

- Allow `--exclude` to be repeated ([#208](https://github.com/opendevstack/tailor/pull/208)).

### Changed

- Improve hint when recreation is prevented ([4e2d04a](https://github.com/opendevstack/tailor/commit/4e2d04a402acb74e925c129dd55b3f31bf2d92e5)).

## [1.1.2] - 2020-06-22

### Fixed

- Do not trim all annotations during export by default. Instead, only a few known annotations are removed (`kubectl.kubernetes.io/last-applied-configuration`, `openshift.io/image.dockerRepositoryCheck`). If you want to trim other annotations, there is a new flag `--trim-annotation`. The flag can be repeated, and allows either exact match or prefix match (e.g. `openshift.io/`). ([#207](https://github.com/opendevstack/tailor/pull/207)).
- Recreate may fail ([#204](https://github.com/opendevstack/tailor/pull/204)).

## [1.1.1] - 2020-06-18

### Fixed

- Create `ServiceAccount` and `RoleBinding` resources before e.g. `DeploymentConfig` resources to avoid pod startup failure due to missing service accounts ([#202](https://github.com/opendevstack/tailor/pull/202)).

## [1.1.0] - 2020-06-16

### Added

- Allow users to selectively apply changes ([#198](https://github.com/opendevstack/tailor/pull/198)).

### Changed

- Use Go 1.14 ([#196](https://github.com/opendevstack/tailor/pull/196)).
- Trim host paths from executable ([#196](https://github.com/opendevstack/tailor/pull/196)).

### Fixed

- Fail with proper error when template parameter does not have `name` property ([#197](https://github.com/opendevstack/tailor/pull/197)).
- Fix panic on invalid template ([#200](https://github.com/opendevstack/tailor/pull/200)).

## [1.0.0] - 2020-05-20

### Added

- Support `Deployment` resource ([#189](https://github.com/opendevstack/tailor/pull/189)).
- Support `LimitRange` and `ResourceQuota` resources ([#194](https://github.com/opendevstack/tailor/pull/194)).
- Export with parameterised namespace by default ([#192](https://github.com/opendevstack/tailor/pull/192)).

### Changed

- Use `oc get --export` instead of deprecated `oc export`. This allows to use Tailor with OpenShift 4 - however there is no manual or automated test setup yet which means OpenShift 4 is not officially supported yet ([#160](https://github.com/opendevstack/tailor/pull/160)).

### Fixed

- Handle missing EOL in params file ([#185](https://github.com/opendevstack/tailor/pull/185)).
- Comparing `CronJob`s works now ([#158](https://github.com/opendevstack/tailor/issues/158)).
- Respect `--exclude` flag when exporting ([#149](https://github.com/opendevstack/tailor/issues/149)).
- Exporting just one resource works as expected now ([#188](https://github.com/opendevstack/tailor/issues/188)).
- Abort generating keys if files are already present ([#193](https://github.com/opendevstack/tailor/pull/193)).

## [0.13.1] - 2020-03-23

### Fixed

- Avoid `--force` flag when run against local cluster ([#182](https://github.com/opendevstack/tailor/pull/182)).

## [0.13.0] - 2020-03-20

### Added

- Prefer `Tailorfile.NAMESPACE` over `Tailorfile`, see [#172](https://github.com/opendevstack/tailor/pull/172).

- Add namespace env file if it exists, see [#173](https://github.com/opendevstack/tailor/pull/173).

- Add `--verify` flag to check if any drift is present after applying ([#179](https://github.com/opendevstack/tailor/pull/179)).

### Changed

- Use `oc apply` instead of generating patches ([#168](https://github.com/opendevstack/tailor/pull/168)).
  The change removes the custom Tailor annotations in favour of
  the `kubectl.kubernetes.io/last-applied-configuration` annotation.
  If there are any issues with unexpected drift or problems when reconciling,
  please consult https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/.

### Fixed

- Exit with 1 when no diff is performed ([#175](https://github.com/opendevstack/tailor/issues/175)).

- Do not calculate drift when adding empty string to missing fields ([#180](https://github.com/opendevstack/tailor/pull/180)).

- Patching of arrays can fail (see [#92](https://github.com/opendevstack/tailor/issues/92).
  This was fixed by using `oc apply` as outlined above.

### Removed

- Remove --context-dir support ([#176](https://github.com/opendevstack/tailor/pull/176)).


## [0.12.0] - 2020-01-31

### Added

- Introduce new flag `--preserve-immutable-fields`. It has the same effect as
  specifying `--preserve` for all immutable fields (such as
  `pvc:/spec/accessModes`). For more background, see [#162](https://github.com/opendevstack/tailor/issues/162).

### Changed

- Rename `status` to `diff` and `update` to `apply`. For backwards compatibility,
  two aliases `status` and `update` have been created. As a consequence, `--diff`
  has been renamed to `--format`. See
  [#163](https://github.com/opendevstack/tailor/pull/163).

- Rename `--ignore-path` to `--preserve` See
  [#165](https://github.com/opendevstack/tailor/pull/165). For backwards
  compatibility, `--ignore-path` is still supported (though deprecated).

- Return with non-zero exit code when detected drift is not reconciled. This
  allows scripts to know whether the user confirmed to apply the changes or not.
  See [#166](https://github.com/opendevstack/tailor/pull/166).

## [0.11.0] - 2019-11-29

### Added
- Support cron jobs resources. See [#155](https://github.com/opendevstack/tailor/issues/155).

## [0.10.4] - 2019-11-25

### Fixed
- Resolve `--param-file` and `--private-key` relative to context directory. See
  [#152](https://github.com/opendevstack/tailor/pull/152) and
  [#153](https://github.com/opendevstack/tailor/pull/153)

## [0.10.3] - 2019-10-18

### Fixed
- Avoid drift in annotations after creation of resource ([#142](https://github.com/opendevstack/tailor/issues/142))
- Remove legacy fields `/userNames` and `/groupNames` from `Rolebinding`
  resources. Those fields are added by `oc export` but might cause issues when
  set to `null`. Tailor removes those now, so only changes in `subjects` have
  any effect. See [#140](https://github.com/opendevstack/tailor/issues/140).
- Do not escape field names when adding annotations object, as when
  `/metadata/annotations` is added, the keys inside need to contain raw slashes
  (`/`) instead of escaped ones (`~1`). Only when annotations already exist,
  keys containing a slash need to escape them. See
  [#145](https://github.com/opendevstack/tailor/issues/145).

## [0.10.2] - 2019-10-09

### Fixed
- Export of resources with `/metadata/generateName` instead of `/metadata/name`
  failed as Tailor expects every resource to have a name. Now Tailor will
  fallback to `generateName`, see [#135](https://github.com/opendevstack/tailor/issues/135).

## [0.10.1] - 2019-10-07

### Fixed
- Deletion of resources does not show notice anymore ([#131](https://github.com/opendevstack/tailor/issues/131))
- Textual diff reflects actual changes corretly now. In 0.10.0, it showed some
  deletions which were not part of the JSON patches. See [#130](https://github.com/opendevstack/tailor/issues/130).

## [0.10.0] - 2019-09-30

### Added

- Support multiple context directories. This basically allows to execute Tailor
  once for multiple directories. Alongside, template and param dir flags have
  been cleaned up. See [#113](https://github.com/opendevstack/tailor/issues/113)
  for more details.

### Changed

- Hide internal annotations ([#112](https://github.com/opendevstack/tailor/issues/122))
- Unify annotation names. You can delete existing annotations in OCP starting
  with `original-values.tailor.io` and
  `managed-annotations.tailor.opendevstack.org`. Note that this is a breaking change: Tailor will show drift on e.g. image references in `DeploymentConfig` resources. Once `tailor update` has run (if the referenced image tag did not change, no de-deployment will happen), the drift disappears. See [#76](https://github.com/opendevstack/tailor/issues/76).
- Avoid drift on empty values to reduce template boilerplate ([#107](https://github.com/opendevstack/tailor/issues/107))
- Hide secrets drift by default. If drift should be shown, pass `--reveal-secrets` ([#109](https://github.com/opendevstack/tailor/issues/109))
- Prevent resource recreation by default. If re-creation should be allowed, pass `--allow-recreate` ([#111](https://github.com/opendevstack/tailor/issues/111))
- Extract OpenShift client to allow easier testing. This should have no effect
  on the tool itself  ([#118](https://github.com/opendevstack/tailor/issues/118))
- Adopt github.com/golang-standards/project-layout. This should have no effect
  on the tool itself.


## [0.9.5] - 2019-07-22

### Added
- Exclusion of kinds/resources/labels via the `--exclude` flag ([#115](https://github.com/opendevstack/tailor/issues/115)).

### Changed
- Addressed issues reported by golangci-lint ([#119](https://github.com/opendevstack/tailor/issues/119), thanks @u5surf).

## [0.9.4] - 2019-04-23

### Fixed 
- Do not issue removal patches when a path with an array value is ignored
- Item is not found when both name and selector is given (oc returns a single item then instead of a list).

## [0.9.3] - 2019-03-15

### Fixed

- `TAILOR_NAMESPACE` detection was not working ([#100](hhttps://github.com/opendevstack/tailor/issues/100)).

## [0.9.2] - 2019-01-18

### Changed
- The `oc` binary might have a different version than the targeted cluster. This
  can lead to incorrect behaviour (e.g. constant drift). To avoid this, Tailor
  now checks whether the client and server version match and refuses to
  continue if this is not the case. Users running Tailor against multiple
  clusters with differing versions can now specify the `oc` binary to use via
  `--oc-binary` or disregard the warning and proceed with `--force`. As a
  side-effect of this change, all operations require an `oc login` now. See 
  [#88](https://github.com/opendevstack/tailor/pull/88).
- In OCP versions above 3.9, a new field `lastTriggeredImage` is added to 
  `imageChangeParams` automatically. The value is not known to template authors
  and consequently stripped from the export before comparison ([#90](https://github.com/opendevstack/tailor/pull/90)).

### Fixed

- Malformed templates caused a panic previously. Now a human-friendly error is
  displayed pointing at the syntax problem ([#85](https://github.com/opendevstack/tailor/pull/85)).

## [0.9.1] - 2018-11-27

### Added

- Automatically pass `TAILOR_NAMESPACE` param to templates if they define it in
  the `parameters` section. The value is set to the namespace that Tailor runs
  against (#77).

### Changed

- Unify log output during update (#78).
- Remove metadata from template as it is unsused by Tailor (#74).
- Ignore selector (e.g. set in a `Tailorfile`) when a specific resource is
  given as Openshift does not accept both selector and resource (#72).
- Surface errors during export as template (#70).

### Fixed

- Remove annotations from exported templates also in case they contain slashes
  (#73).


## 0.9.0 (2018-10-29)

* [Feature] Rewrite diff engine. Tailor now uses `oc patch` under the hood to
  to update resources (instead of `oc apply`). The new behaviour is more correct
  and less dangerous. (#49)
* [Feature] Warn before all resources are deleted. `--force` has to be given to
  proceed in this case. (#66)
* [Feature] Implement flag to ignore paths, `--ignore-path`. Paths can be
  ignored for all kinds, for one kind or for one resource. (#63)
* [Feature] Add change summary at the end of `status` and `update` commands.
* [Feature] Support multiple param files via repearing `--param-file`. (#56)
* [Feature] Check if OCP project exists during initialisation. (#55)
* [Fix] Apply resource changes in specific order, starting with resources that
  do not have dependencies. (#54)
* [Task] Treat creationTimestamp as platform managed field (#51)
* [Task] Switch to Go 1.11 modules (#52)

## 0.8.0 (2018-08-01)

* [Feature] Implement general handling of immutable fields. Previously Tailor tried to modify the resource which resulted in an error, now it recreates the resource (#33).
* [Feature] Add `--debug` flag, which contains debug information, reducing the amount of printed information when `--verbose` is given. `--debug` implies `--verbose`.
* [Fix] Image reference changed accidentally when other parts of the DeploymentConfig changed (#38).
* [Fix] Propagate exit code from `oc` properly when running `tailor update`.

## 0.7.0 (2018-07-30)

* [Feature] Support multiple labels (#7).
* [Feature] Allow comments in `Tailorfile` (either `#` or `//`).
* [Fix] Allow whitespace around flags and their arguments in `Tailorfile`.
* [Fix] Check param dir existance if explicitly given (#24).
* [Fix] Don't panic when params are duplicated (#12).
* [Fix] Prevent original values annotation in exports (#3 and #5).
* [Task] Improve text about resource limiting (#25).
* [Task] Improve diff description (#6).
* [Task] Show found resources (#14).

## 0.6.1 (2018-07-26)

* [Fix] Respect flags defined in Tailorfile that have default values (#22).

## 0.6.0 (2018-07-25)

* [Feature] `Tailorfile` support.
* [Task] Use compact help template (which allows to show help for just one command).
* [Fix] Fix wrong export target when just one resource is specified (#11).
* [Fix] Percent signs (%) are not automatically encoded (duplicated) to prevent issues (#4).
* [Fix] Error when deleting resource due to `--selector` not allowed during delete operation (#13).

## 0.5.1 (2018-07-12)

* Fix exiting with 1 and error statement all the time even though everything went fine.

## 0.5.0 (2018-07-12)

* Rename `ocdiff` to `tailor` and release on GitHub.

## 0.4.0

* Prevent rendering when no public key files are found.
* Check if file exists at beginning of reveal command.
* Proper error message if editor is not found in PATH.
* Fix wrong export when a single resource was targeted.
* Prefer folder `public-keys` over working dir if present when looking for public keys.
* Increase specifity of blacklisted key regexes.
* Prefer folder `<namespace>`` over working dir if present when looking for param files.
* Deal with modified values by saving the original value in an annotation and resetting the key to that value before comparing.

## 0.3.0

* Compile and compare single list instead of multiple lists. This reduces the number of API calls required and allows to export properly.

## 0.2.0

* Secrets handling (PGP encryption)

## 0.1.0

Initial version
