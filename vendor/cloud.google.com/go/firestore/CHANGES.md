# Changes

## [1.20.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.19.0...firestore/v1.20.0) (2025-10-20)


### Features

* **firestore:** Add omitzero struct tag option ([#12992](https://github.com/googleapis/google-cloud-go/issues/12992)) ([c2f9c74](https://github.com/googleapis/google-cloud-go/commit/c2f9c7408f0a1c003da19db5520da8a745474f38))


### Bug Fixes

* **firestore:** Handle unused values ([#13103](https://github.com/googleapis/google-cloud-go/issues/13103)) ([a109cf6](https://github.com/googleapis/google-cloud-go/commit/a109cf6bde9675f8bc5edba61dce49f4852709d6)), refs [#9784](https://github.com/googleapis/google-cloud-go/issues/9784)
* **firestore:** Upgrade gRPC service registration func ([8fffca2](https://github.com/googleapis/google-cloud-go/commit/8fffca2819fa3dc858c213aa0c503e0df331b084))

## [1.19.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.18.0...firestore/v1.19.0) (2025-10-06)


### Features

* **firestore/apiv1:** Add Firestore CloneDatabase support ([#12629](https://github.com/googleapis/google-cloud-go/issues/12629)) ([0b74f43](https://github.com/googleapis/google-cloud-go/commit/0b74f43c6639e0e85c828145e867b0e98f4fdc96))
* **firestore/apiv1:** Expose tags field in Database and RestoreDatabaseRequest public protos ([f3c3c1a](https://github.com/googleapis/google-cloud-go/commit/f3c3c1ace2e36aa5e5c4c73e39a42cf3fbe2dbcb))
* **firestore:** Add FieldNotFoundError for missing fields ([#12899](https://github.com/googleapis/google-cloud-go/issues/12899)) ([4df2b74](https://github.com/googleapis/google-cloud-go/commit/4df2b74208a851db0e0aeeef8f7e86c2d450e102))
* **firestore:** Allow creating REST clients ([#12575](https://github.com/googleapis/google-cloud-go/issues/12575)) ([bfd138a](https://github.com/googleapis/google-cloud-go/commit/bfd138a71ffb4f230f5f9bd28f97379b36c143a9))
* **firestore:** Expose previous state of document ([#12894](https://github.com/googleapis/google-cloud-go/issues/12894)) ([5c4524a](https://github.com/googleapis/google-cloud-go/commit/5c4524a209f78d82c3a1b3fe8beec247a67bceb0))
* **firestore:** Introduce DocFromResourceName for full resource paths ([#12307](https://github.com/googleapis/google-cloud-go/issues/12307)) ([f7ee0b2](https://github.com/googleapis/google-cloud-go/commit/f7ee0b25a69336b290d47c19657c2abdbfec711f))


### Bug Fixes

* **firestore:** Add dedicated lock to BulkWriter to prevent race ([#12896](https://github.com/googleapis/google-cloud-go/issues/12896)) ([beb7d97](https://github.com/googleapis/google-cloud-go/commit/beb7d97545c6a55b873f0232f4d0710ee4c9d030))
* **firestore:** Correct ReadWrite transaction retries ([#12893](https://github.com/googleapis/google-cloud-go/issues/12893)) ([b7124db](https://github.com/googleapis/google-cloud-go/commit/b7124db194505c4fc124491c7f8194bb526806f2))
* **firestore:** Initialize readSettings in queries to prevent panic ([#12898](https://github.com/googleapis/google-cloud-go/issues/12898)) ([18bce36](https://github.com/googleapis/google-cloud-go/commit/18bce369f352554f450b44f13deb8c15b236f02f)), refs [#12448](https://github.com/googleapis/google-cloud-go/issues/12448)

## [1.18.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.17.0...firestore/v1.18.0) (2025-01-02)


### Features

* **firestore:** Add String method for Update struct ([#11355](https://github.com/googleapis/google-cloud-go/issues/11355)) ([2320c35](https://github.com/googleapis/google-cloud-go/commit/2320c35ad9a7244c992bfe528e8d49fdc4089369))
* **firestore:** Add WithCommitResponseTo TransactionOption ([#6967](https://github.com/googleapis/google-cloud-go/issues/6967)) ([eb25266](https://github.com/googleapis/google-cloud-go/commit/eb252663ad0bdabbd5de1767b42a69fd2aee54b2))
* **firestore:** Surfacing the error returned from the service in Bulkwriter ([#10826](https://github.com/googleapis/google-cloud-go/issues/10826)) ([9ae039a](https://github.com/googleapis/google-cloud-go/commit/9ae039a38856133a2bde4c3bd70155d17538c974))


### Bug Fixes

* **firestore:** Add UTF-8 validation ([#10881](https://github.com/googleapis/google-cloud-go/issues/10881)) ([9199843](https://github.com/googleapis/google-cloud-go/commit/9199843947bc3a0fa415dba50ba2221850e0fbad))
* **firestore:** Allow using != with nil ([#11112](https://github.com/googleapis/google-cloud-go/issues/11112)) ([5b59819](https://github.com/googleapis/google-cloud-go/commit/5b59819e2d603ef55c4cf056b70af6a08d335373))
* **firestore:** Update golang.org/x/net to v0.33.0 ([e9b0b69](https://github.com/googleapis/google-cloud-go/commit/e9b0b69644ea5b276cacff0a707e8a5e87efafc9))
* **firestore:** Update google.golang.org/api to v0.203.0 ([8bb87d5](https://github.com/googleapis/google-cloud-go/commit/8bb87d56af1cba736e0fe243979723e747e5e11e))
* **firestore:** WARNING: On approximately Dec 1, 2024, an update to Protobuf will change service registration function signatures to use an interface instead of a concrete type in generated .pb.go files. This change is expected to affect very few if any users of this client library. For more information, see https://togithub.com/googleapis/google-cloud-go/issues/11020. ([8bb87d5](https://github.com/googleapis/google-cloud-go/commit/8bb87d56af1cba736e0fe243979723e747e5e11e))

## [1.17.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.16.0...firestore/v1.17.0) (2024-09-11)


### Features

* **firestore/apiv1:** Add Database.CmekConfig and Database.cmek_config (information about CMEK enablement) ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Add Database.delete_time (the time a database was deleted, if it ever was) ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Add Database.previous_id (if a database was deleted, what ID it was using beforehand) ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Add Database.SourceInfo and Database.source_info (information about database provenance, specifically for restored databases) ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Allow specifying an encryption_config when restoring a database ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore:** Add support for Go 1.23 iterators ([84461c0](https://github.com/googleapis/google-cloud-go/commit/84461c0ba464ec2f951987ba60030e37c8a8fc18))
* **firestore:** Expose the `FindNearest.distance_result_field` parameter ([9a5144e](https://github.com/googleapis/google-cloud-go/commit/9a5144e7d30c6f058b13fdf3fd9436904e77dff0))
* **firestore:** Expose the `FindNearest.distance_threshold` parameter ([9a5144e](https://github.com/googleapis/google-cloud-go/commit/9a5144e7d30c6f058b13fdf3fd9436904e77dff0))
* **firestore:** Query profiling ([#10164](https://github.com/googleapis/google-cloud-go/issues/10164)) ([58052a2](https://github.com/googleapis/google-cloud-go/commit/58052a2eefd56b3129e04f177398b3ffb688d4d7))


### Bug Fixes

* **firestore:** Bump dependencies ([2ddeb15](https://github.com/googleapis/google-cloud-go/commit/2ddeb1544a53188a7592046b98913982f1b0cf04))
* **firestore:** Retry batchwrite only on retryable errors ([#10603](https://github.com/googleapis/google-cloud-go/issues/10603)) ([23e5df5](https://github.com/googleapis/google-cloud-go/commit/23e5df5b8ee40317ab0d1ac6bb2b92ccc054426c))
* **firestore:** Update google.golang.org/api to v0.191.0 ([5b32644](https://github.com/googleapis/google-cloud-go/commit/5b32644eb82eb6bd6021f80b4fad471c60fb9d73))


### Documentation

* **firestore/apiv1:** Clarify maximum retention of backups (max 14 weeks) ([2710d0f](https://github.com/googleapis/google-cloud-go/commit/2710d0f8c66c17f1ddb1d4cc287f7aeb701c0f72))
* **firestore/apiv1:** Clarify restore details ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Fix assorted capitalization issues with the word "ID" ([2d5a9f9](https://github.com/googleapis/google-cloud-go/commit/2d5a9f9ea9a31e341f9a380ae50a650d48c29e99))
* **firestore/apiv1:** Remove note about backups running at a specific time ([2710d0f](https://github.com/googleapis/google-cloud-go/commit/2710d0f8c66c17f1ddb1d4cc287f7aeb701c0f72))
* **firestore/apiv1:** Standardize on the capitalization of "ID" ([2710d0f](https://github.com/googleapis/google-cloud-go/commit/2710d0f8c66c17f1ddb1d4cc287f7aeb701c0f72))
* **firestore:** Minor documentation clarifications on FindNearest DistanceMeasure options ([5b4b0f7](https://github.com/googleapis/google-cloud-go/commit/5b4b0f7878276ab5709011778b1b4a6ffd30a60b))

## [1.16.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.15.0...firestore/v1.16.0) (2024-07-24)


### Features

* **firestore/apiv1:** Add bulk delete api ([#10369](https://github.com/googleapis/google-cloud-go/issues/10369)) ([134f567](https://github.com/googleapis/google-cloud-go/commit/134f567c18892d6050f60ae875a3de7738104da0))
* **firestore/apiv1:** Add Vector Index API ([f8ff971](https://github.com/googleapis/google-cloud-go/commit/f8ff971366999aefb5eb5189c6c9e2bd76a05d9e))
* **firestore:** Adding vector search ([#10548](https://github.com/googleapis/google-cloud-go/issues/10548)) ([5c0d6df](https://github.com/googleapis/google-cloud-go/commit/5c0d6df5cc28659c5fbd54329f8b6c134cf95730))


### Bug Fixes

* **firestore:** Bump google.golang.org/api@v0.187.0 ([8fa9e39](https://github.com/googleapis/google-cloud-go/commit/8fa9e398e512fd8533fd49060371e61b5725a85b))
* **firestore:** Bump google.golang.org/grpc@v1.64.1 ([8ecc4e9](https://github.com/googleapis/google-cloud-go/commit/8ecc4e9622e5bbe9b90384d5848ab816027226c5))
* **firestore:** Bump x/net to v0.24.0 ([ba31ed5](https://github.com/googleapis/google-cloud-go/commit/ba31ed5fda2c9664f2e1cf972469295e63deb5b4))
* **firestore:** Move createIndexes calls ([#9714](https://github.com/googleapis/google-cloud-go/issues/9714)) ([d931626](https://github.com/googleapis/google-cloud-go/commit/d9316263ca4ad0667d4a0f886a4977b36585b572))
* **firestore:** Update dependencies ([257c40b](https://github.com/googleapis/google-cloud-go/commit/257c40bd6d7e59730017cf32bda8823d7a232758))


### Documentation

* **firestore/apiv1:** Allow 14 week backup retention for Firestore daily backups ([#9685](https://github.com/googleapis/google-cloud-go/issues/9685)) ([2cdc40a](https://github.com/googleapis/google-cloud-go/commit/2cdc40a0b4288f5ab5f2b2b8f5c1d6453a9c81ec))
* **firestore/apiv1:** Correct BackupSchedule recurrence docs that mentioned specific time of day ([fe85be0](https://github.com/googleapis/google-cloud-go/commit/fe85be03d1e6ba69182ff1045a3faed15aa00128))
* **firestore/apiv1:** Update field api description ([134f567](https://github.com/googleapis/google-cloud-go/commit/134f567c18892d6050f60ae875a3de7738104da0))

## [1.15.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.14.0...firestore/v1.15.0) (2024-03-05)


### Features

* **firestore/apiv1:** Add DeleteDatabase API and delete protection ([#9185](https://github.com/googleapis/google-cloud-go/issues/9185)) ([ec9b526](https://github.com/googleapis/google-cloud-go/commit/ec9b5268627734c40efd15353cf4bc83a837ff3a))
* **firestore/apiv1:** Expose Firestore PITR fields in Database to stable ([5132d0f](https://github.com/googleapis/google-cloud-go/commit/5132d0fea3a5ac902a2c9eee865241ed4509a5f4))
* **firestore:** Add new types QueryMode, QueryPlan, ResultSetStats ([82054d0](https://github.com/googleapis/google-cloud-go/commit/82054d0e6905358e48517cbc8ea844dfb624082c))


### Bug Fixes

* **firestore:** Bump google.golang.org/api to v0.149.0 ([8d2ab9f](https://github.com/googleapis/google-cloud-go/commit/8d2ab9f320a86c1c0fab90513fc05861561d0880))
* **firestore:** Correct the cursors when LimitToLast is used ([#9413](https://github.com/googleapis/google-cloud-go/issues/9413)) ([2090651](https://github.com/googleapis/google-cloud-go/commit/2090651b4a7a1dc3be5af4e7ac4607fbc3ffccac))
* **firestore:** Enable universe domain resolution options ([fd1d569](https://github.com/googleapis/google-cloud-go/commit/fd1d56930fa8a747be35a224611f4797b8aeb698))
* **firestore:** Remove types QueryMode, QueryPlan, ResultSetStats ([97d62c7](https://github.com/googleapis/google-cloud-go/commit/97d62c7a6a305c47670ea9c147edc444f4bf8620))
* **firestore:** Return status code from bulkwriter results ([#9030](https://github.com/googleapis/google-cloud-go/issues/9030)) ([e8223c6](https://github.com/googleapis/google-cloud-go/commit/e8223c6ee544237b54b351e421b7092dc3b237a6))
* **firestore:** Update grpc-go to v1.56.3 ([343cea8](https://github.com/googleapis/google-cloud-go/commit/343cea8c43b1e31ae21ad50ad31d3b0b60143f8c))
* **firestore:** Update grpc-go to v1.59.0 ([81a97b0](https://github.com/googleapis/google-cloud-go/commit/81a97b06cb28b25432e4ece595c55a9857e960b7))


### Documentation

* **firestore/apiv1:** Fix formatting due to unclosed backtick ([0500c7a](https://github.com/googleapis/google-cloud-go/commit/0500c7a7f9a9e8629a091558fa258ca7c5028474))

## [1.14.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.13.0...firestore/v1.14.0) (2023-10-19)


### Features

* **firestore:** SUM and AVG aggregations ([#8293](https://github.com/googleapis/google-cloud-go/issues/8293)) ([011f9ff](https://github.com/googleapis/google-cloud-go/commit/011f9ff083bebad5c30443b3b0fd9df68579a65b))


### Bug Fixes

* **firestore:** Update golang.org/x/net to v0.17.0 ([174da47](https://github.com/googleapis/google-cloud-go/commit/174da47254fefb12921bbfc65b7829a453af6f5d))

## [1.13.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.12.0...firestore/v1.13.0) (2023-09-18)


### Features

* **firestore:** Support for multiple databases ([#5331](https://github.com/googleapis/google-cloud-go/issues/5331)) ([94d4b1b](https://github.com/googleapis/google-cloud-go/commit/94d4b1b58d2c8f3dac18e7efb0be641b6311c775))


### Bug Fixes

* **firestore:** Compare full resource path when docs ordered by __name__ ([#8409](https://github.com/googleapis/google-cloud-go/issues/8409)) ([5ef93de](https://github.com/googleapis/google-cloud-go/commit/5ef93de226b854bdf6277b7f906b86755a07d229))
* **firestore:** Correcting EndBefore with LimitToLast behaviour ([#8370](https://github.com/googleapis/google-cloud-go/issues/8370)) ([350f7ad](https://github.com/googleapis/google-cloud-go/commit/350f7adb2a087811a70f1c05bf71014022aefeb4))

## [1.12.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.11.0...firestore/v1.12.0) (2023-08-01)


### Features

* **firestore:** Publish proto definitions for SUM/AVG in Firestore ([e3f8c89](https://github.com/googleapis/google-cloud-go/commit/e3f8c89429a207c05fee36d5d93efe76f9e29efe))

## [1.11.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.10.0...firestore/v1.11.0) (2023-06-26)


### Features

* **firestore:** Update all direct dependencies ([b340d03](https://github.com/googleapis/google-cloud-go/commit/b340d030f2b52a4ce48846ce63984b28583abde6))


### Bug Fixes

* **firestore:** Cleanup integration test resources ([#8057](https://github.com/googleapis/google-cloud-go/issues/8057)) ([210584d](https://github.com/googleapis/google-cloud-go/commit/210584df494e9627dd13c24138fcbebe85048647))
* **firestore:** Do not trace iterator.Done error ([#8082](https://github.com/googleapis/google-cloud-go/issues/8082)) ([5f24d17](https://github.com/googleapis/google-cloud-go/commit/5f24d173db35358d241de186953cd094dae312c9)), refs [#7711](https://github.com/googleapis/google-cloud-go/issues/7711)
* **firestore:** REST query UpdateMask bug ([df52820](https://github.com/googleapis/google-cloud-go/commit/df52820b0e7721954809a8aa8700b93c5662dc9b))

## [1.10.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.9.0...firestore/v1.10.0) (2023-05-22)


### Features

* **firestore:** Add `OR` query support docs: Improve the API documentation for the `Firestore.ListDocuments` RPC docs: Minor documentation formatting and cleanup ([aeb6fec](https://github.com/googleapis/google-cloud-go/commit/aeb6fecc7fd3f088ff461a0c068ceb9a7ae7b2a3))
* **firestore:** Add bloom filter related proto fields PiperOrigin-RevId: 529511263 ([31c3766](https://github.com/googleapis/google-cloud-go/commit/31c3766c9c4cab411669c14fc1a30bd6d2e3f2dd))
* **firestore:** Add REST client ([06a54a1](https://github.com/googleapis/google-cloud-go/commit/06a54a16a5866cce966547c51e203b9e09a25bc0))
* **firestore:** Added support for REST transport ([aeb6fec](https://github.com/googleapis/google-cloud-go/commit/aeb6fecc7fd3f088ff461a0c068ceb9a7ae7b2a3))
* **firestore:** EntityFilter for AND/OR queries ([#7757](https://github.com/googleapis/google-cloud-go/issues/7757)) ([ae37793](https://github.com/googleapis/google-cloud-go/commit/ae377932de20d99f31766ca1cccd2d1cfa18a1c0))
* **firestore:** Rewrite signatures and type in terms of new location ([620e6d8](https://github.com/googleapis/google-cloud-go/commit/620e6d828ad8641663ae351bfccfe46281e817ad))
* **firestore:** Update iam and longrunning deps ([91a1f78](https://github.com/googleapis/google-cloud-go/commit/91a1f784a109da70f63b96414bba8a9b4254cddd))


### Bug Fixes

* **firestore:** Enable rest_numeric_enums for PHP client ([2fef56f](https://github.com/googleapis/google-cloud-go/commit/2fef56f75a63dc4ff6e0eea56c7b26d4831c8e27))
* **firestore:** Replace usage of transform with update_transform in batch write  ([#7864](https://github.com/googleapis/google-cloud-go/issues/7864)) ([949e4d8](https://github.com/googleapis/google-cloud-go/commit/949e4d8001040e78f2ad9b1e5cbf5b9113d8f3ef))
* **firestore:** Update grpc to v1.55.0 ([1147ce0](https://github.com/googleapis/google-cloud-go/commit/1147ce02a990276ca4f8ab7a1ab65c14da4450ef))

## [1.9.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.8.0...firestore/v1.9.0) (2022-11-29)


### Features

* **firestore:** start generating proto stubs ([eed371e](https://github.com/googleapis/google-cloud-go/commit/eed371e9b1639c81663c6858db119fb87a126454))


### Documentation

* **firestore:** Adds emulator snippet ([#6926](https://github.com/googleapis/google-cloud-go/issues/6926)) ([456afab](https://github.com/googleapis/google-cloud-go/commit/456afab76f078ef58b7e5b3409acc6b3f71c5b79))

## [1.8.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.7.0...firestore/v1.8.0) (2022-10-17)


### Features

* **firestore:** Adds COUNT aggregation query ([#6692](https://github.com/googleapis/google-cloud-go/issues/6692)) ([31ac692](https://github.com/googleapis/google-cloud-go/commit/31ac692d925065981a695266d1e4e22e5374725e))
* **firestore:** Adds snapshot reads impl. ([#6718](https://github.com/googleapis/google-cloud-go/issues/6718)) ([43cc5bc](https://github.com/googleapis/google-cloud-go/commit/43cc5bc068d2f3abdde6c65beaac349218fc1a02))

## [1.7.0](https://github.com/googleapis/google-cloud-go/compare/firestore/v1.6.1...firestore/v1.7.0) (2022-10-06)


### Features

* **firestore/apiv1:** add firestore aggregation query apis to the stable googleapis branch ([ec1a190](https://github.com/googleapis/google-cloud-go/commit/ec1a190abbc4436fcaeaa1421c7d9df624042752))
* **firestore:** Adds Bulkwriter support to Firestore client ([#5946](https://github.com/googleapis/google-cloud-go/issues/5946)) ([20b6c1b](https://github.com/googleapis/google-cloud-go/commit/20b6c1bbbc28311f4388e163cd9358d1ac0e94d4))
* **firestore:** expose read_time fields in Firestore PartitionQuery and ListCollectionIds, currently only available in private preview ([90489b1](https://github.com/googleapis/google-cloud-go/commit/90489b10fd7da4cfafe326e00d1f4d81570147f7))

### [1.6.1](https://www.github.com/googleapis/google-cloud-go/compare/firestore/v1.6.0...firestore/v1.6.1) (2021-10-29)


### Bug Fixes

* **firestore:** prefer exact matches when reflecting fields ([#4908](https://www.github.com/googleapis/google-cloud-go/issues/4908)) ([d3d9420](https://www.github.com/googleapis/google-cloud-go/commit/d3d94205995ad910bd277f1f930cef4ac86c8040))

## [1.6.0](https://www.github.com/googleapis/google-cloud-go/compare/firestore/v1.5.0...firestore/v1.6.0) (2021-09-09)


### Features

* **firestore:** Add support for PartitionQuery ([#4206](https://www.github.com/googleapis/google-cloud-go/issues/4206)) ([b34783a](https://www.github.com/googleapis/google-cloud-go/commit/b34783a4d7a8c88204e0f44bd411795d8267d811))
* **firestore:** Support DocumentRefs in OrderBy, Add Query.Serialize, Query.Deserialize for cross machine serialization ([#4347](https://www.github.com/googleapis/google-cloud-go/issues/4347)) ([a0f7a02](https://www.github.com/googleapis/google-cloud-go/commit/a0f7a02bd8db90fa2297c6e84658868901ef9566))


### Bug Fixes

* **firestore:** correct an issue with returning empty paritions from GetPartionedQueries ([#4346](https://www.github.com/googleapis/google-cloud-go/issues/4346)) ([b2a6171](https://www.github.com/googleapis/google-cloud-go/commit/b2a61719b3caf43b095fc290b23de245a2135512))
* **firestore:** remove excessive spans on iterator ([#4163](https://www.github.com/googleapis/google-cloud-go/issues/4163)) ([812ef1f](https://www.github.com/googleapis/google-cloud-go/commit/812ef1ffdce2e87570660b58f0e725ad51f68546))
* **firestore:** retry RESOURCE_EXHAUSTED errors docs: various documentation improvements ([9a459d5](https://www.github.com/googleapis/google-cloud-go/commit/9a459d5d149b9c3b02a35d4245d164b899ff09b3))

## [1.5.0](https://www.github.com/googleapis/google-cloud-go/compare/v1.4.0...v1.5.0) (2021-02-24)


### Features

* **firestore:** add opencensus tracing support  ([#2942](https://www.github.com/googleapis/google-cloud-go/issues/2942)) ([257f322](https://www.github.com/googleapis/google-cloud-go/commit/257f322e68b75765bd316ccefed5461d4df538a0))


### Bug Fixes

* **firestore:** address a missing branch in watch.stop() error remapping ([#3643](https://www.github.com/googleapis/google-cloud-go/issues/3643)) ([89ad55d](https://www.github.com/googleapis/google-cloud-go/commit/89ad55d72f79995a68f9c2ed1cd9b5ba50009d6d))

## [1.4.0](https://www.github.com/googleapis/google-cloud-go/compare/firestore/v1.3.0...v1.4.0) (2020-12-03)


### Features

* **firestore:** support "!=" and "not-in" query operators ([#3207](https://www.github.com/googleapis/google-cloud-go/issues/3207)) ([5c44019](https://www.github.com/googleapis/google-cloud-go/commit/5c440192105fe3e9b5dd1b584118b309113935e3)), closes [/firebase.google.com/support/release-notes/js#version_7210_-_september_17_2020](https://www.github.com/googleapis//firebase.google.com/support/release-notes/js/issues/version_7210_-_september_17_2020)

## v1.3.0

- Add support for LimitToLast feature for queries. This allows
  a query to return the final N results. See docs
  [here](https://firebase.google.com/docs/reference/js/firebase.database.Query#limittolast).
- Add support for FieldTransformMinimum and FieldTransformMaximum.
- Add exported SetGoogleClientInfo method.
- Various updates to autogenerated clients.

## v1.2.0

- Deprecate v1beta1 client.
- Fix serverTimestamp docs.
- Add missing operators to query docs.
- Make document IDs 20 alpha-numeric characters. Previously, there could be more
  than 20 non-alphanumeric characters, which broke some users. See
  https://github.com/googleapis/google-cloud-go/issues/1715.
- Various updates to autogenerated clients.

## v1.1.1

- Fix bug in CollectionGroup query validation.

## v1.1.0

- Add support for `in` and `array-contains-any` query operators.

## v1.0.0

This is the first tag to carve out firestore as its own module. See:
https://github.com/golang/go/wiki/Modules#is-it-possible-to-add-a-module-to-a-multi-module-repository.
