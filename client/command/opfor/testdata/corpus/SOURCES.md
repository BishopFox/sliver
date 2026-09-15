# SliverArmory CNA corpus

These unmodified CNA scripts are pinned regression fixtures for the OPFOR client integration. The tests pair them with inert object bytes and verify script loading, command registration, resource resolution, BOF argument packing, and Sliver dispatch; they do not execute the embedded BOF code locally.

| Fixture | Upstream revision | License |
| --- | --- | --- |
| `firefoxdump/firefoxdump.cna` | [`sliverarmory/firefoxdump@19bec6f`](https://github.com/sliverarmory/firefoxdump/tree/19bec6fb2def510b299430955fb2be79c1f51820) | GPL-3.0-only; see the Sliver repository root `LICENSE` |
| `bof_collection/cat.cna` | [`sliverarmory/bof_collection@d937d25`](https://github.com/sliverarmory/bof_collection/tree/d937d25a164f40192e289b67e9c550d2bd4c369a) | Apache-2.0; copied `LICENSE` and `NOTICE` are retained here |
| `operatorskit/finddotnet.cna` and `findsysmon.cna` | [`sliverarmory/OperatorsKit@66368f4`](https://github.com/sliverarmory/OperatorsKit/tree/66368f4738528d26cc1ccc6d9a3c93d44d63edc1) | MIT; copied `LICENSE` is retained here |

The object files are intentionally not vendored. Live BOF execution belongs in a safety-gated Windows integration environment, while these fixtures keep the client test suite deterministic and platform independent.
