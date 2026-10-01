# Example input folders

One input folder per profile, ready to check or build. Each folder follows the
[input specification](../docs/input-spec.md); the metadata is invented.

| Folder | Profile | What it shows |
|---|---|---|
| [`basic/`](basic/) | `basic` | `description.csv` with Meemoo's keys, in Dutch and English |
| [`eark/`](eark/) | `eark` | `description.csv` with Dublin Core keys, a description of the representation, received PREMIS, `representations.csv` |
| [`eark-mods/`](eark-mods/) | `eark-mods` | a finished `mods.xml` with two physical copies, plus the same extras as `eark/` |

Each package has one representation, `master`, with one small JPEG and a
documentation file.

```sh
./bin/sip-creator check --profile eark examples/eark
./bin/sip-creator create --profile eark examples/eark sip-out
```

`create` needs `SIP_SUBMITTER_NAME` set, and `basic` also needs `SIP_SUBMITTER_OR_ID`
(see [Configuration](../README.md#configuration)). The folders hold no
`siegfried.json`, so the packages carry no format info unless you generate one
(see [Format characterization](../README.md#format-characterization)); generate it in a
copy, not in this folder.
