# Changelog

## Unreleased

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
