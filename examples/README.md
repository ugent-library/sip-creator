# Example input folders

One input folder per profile, ready to check or build, at the profile's name. Each
folder follows the [input specification](../docs/input-spec.md); the metadata is
invented.

| Folder | Profile | What it shows |
|---|---|---|
| [`meemoo/basic/`](meemoo/basic/) | `meemoo/basic` | `description.csv` with Meemoo's keys, in Dutch and English |
| [`ugent/basic/`](ugent/basic/) | `ugent/basic` | `description.csv` with Dublin Core keys, a description of the representation, received PREMIS, `representations.csv` |
| [`ugent/bibliographic/`](ugent/bibliographic/) | `ugent/bibliographic` | a finished `mods.xml` with two physical copies, plus the same extras as `ugent/basic/` |

Each package has one representation, `master`, with one small JPEG and a
documentation file.

```sh
./bin/sip-creator check --profile ugent/basic examples/ugent/basic
./bin/sip-creator create --profile ugent/basic examples/ugent/basic sip-out
```

`check` lists any problems, then summarizes what it read. For `ugent/basic/`:

```
Input folder:         examples/ugent/basic
Profile:              ugent/basic

Descriptive metadata: description.csv
Representations:      1 (1 with its own description)
Essence files:        1
Documentation files:  2
PREMIS files:         2
Format report:        not supplied (files carry no format information)

OK: the folder meets the input specification for profile ugent/basic.
```

`create` needs `SIP_SUBMITTER_NAME` set, and `meemoo/basic` also needs
`SIP_SUBMITTER_OR_ID` (see [Configuration](../README.md#configuration)). The folders
hold no `siegfried.json`, so the packages carry no format info unless you generate one
(see [Format characterization](../README.md#format-characterization)); generate it in a
copy, not in this folder.
