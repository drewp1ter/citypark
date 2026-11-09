# Repository Guidelines

## Project Structure & Module Organization

`booking/` contains the XML export of the 1C:Enterprise configuration `УправлениеГостиницей` (SORC hotel management), within the parent City Park Git repository.

- `Configuration.xml` defines the configuration; `ConfigDumpInfo.xml` tracks exported objects.
- `Catalogs/`, `Documents/`, `InformationRegisters/`, and `AccumulationRegisters/` contain domain metadata and object modules.
- `CommonModules/*/Ext/Module.bsl` contains shared BSL logic; `Ext/` contains configuration-level modules.
- `CommonForms/`, object `Forms/`, `CommonPictures/`, and `CommonTemplates/` contain UI definitions and assets.
- `Reports/`, `DataProcessors/`, `Roles/`, and service directories contain reporting, processing, permissions, and integrations.
- `.codex/skills/` provides specialized editing and validation workflows.

## Build, Test, and Development Commands

There is no local npm or Make build entry point. Development requires an installed 1C platform and a development information base.

1. In 1C Designer, load the configuration from files using this directory as the source.
2. Check the configuration and update the development database configuration.
3. Start 1C:Enterprise against that database to exercise the changed behavior.
4. Export changes back to files and review the resulting diff.

Run `git diff -- .` from `booking/` to review this subtree and `git diff --check -- .` to detect whitespace errors. Consult `.codex/skills/db-load-xml/SKILL.md` and `db-run/SKILL.md` before using their automation.

## Coding Style & Naming Conventions

Follow surrounding Russian BSL keywords and descriptive Russian identifiers, such as `ОбработкаПроведения`. Use tabs to match existing indentation. Preserve each file’s encoding, BOM, and line endings. Retain generated XML namespaces, UUIDs, and object references; avoid unrelated reformatting. No standalone formatter or linter configuration was found in this directory.

## Testing Guidelines

No project test suite or coverage threshold was found. Validate changed metadata with the relevant local validation skill and check BSL in Designer. Manually exercise affected forms, document posting, register movements, and access roles as applicable. Record the platform version, scenarios, and results; XML validation alone does not verify runtime behavior.

## Commit & Pull Request Guidelines

Parent history includes `task NN: <title>`, `docs: <summary>`, and `test: <summary>`. Use numbered commits for assigned numbered tasks; keep changes focused. PRs should explain behavior changes, link relevant issues, list validation performed, and include screenshots for UI changes. Describe any required database update. Keep credentials and information-base dumps out of commits.
