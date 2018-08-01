# Changelog

## Unreleased

* [Feature] Implement general handling of immutable fields. Previously Tailor tried to modify the resource which resulted in an error, now it tries to recreate the resource. See #33.
* [Feature] `--debug` flag, which contains debug information, reducing the amount of printed information when `--verbose` is given.
* [Fix] Image changed accidentally when other parts of the DeploymentConfig changed (#38).
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
