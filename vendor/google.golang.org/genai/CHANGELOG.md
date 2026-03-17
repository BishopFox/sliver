# Changelog

## [1.49.0](https://github.com/googleapis/go-genai/compare/v1.48.0...v1.49.0) (2026-02-28)


### Features

* Update data types from discovery doc. ([37134ea](https://github.com/googleapis/go-genai/commit/37134ea8c0c17d262c26ed46e31ada75406dff02))

## [1.48.0](https://github.com/googleapis/go-genai/compare/v1.47.0...v1.48.0) (2026-02-25)


### Features

* Add Image Grounding support to GoogleSearch tool ([ba34adf](https://github.com/googleapis/go-genai/commit/ba34adf470dbb213383df5951ee641cd899958ef))
* enable server side MCP and disable all other AFC when server side MCP is configured. ([a550b3c](https://github.com/googleapis/go-genai/commit/a550b3c7f84e0fe2fd5e5221ac3aa7d20ce4cbf2))

## [1.47.0](https://github.com/googleapis/go-genai/compare/v1.46.0...v1.47.0) (2026-02-18)


### Features

* Support multimodal embedding for Gemini Embedding 2.0 and support MaaS models in Models.embed_content() (Vertex AI API) ([ef61dd1](https://github.com/googleapis/go-genai/commit/ef61dd1f3c65e185c594d07a2cf928e14f3b04ec))

## [1.46.0](https://github.com/googleapis/go-genai/compare/v1.45.0...v1.46.0) (2026-02-09)


### Features

* Support encryption_spec in tuning job creation configuration for GenAI SDK ([025a3f0](https://github.com/googleapis/go-genai/commit/025a3f0c8d88c95ebb35f884e35d389cfdd7affa))


### Bug Fixes

* round up timeout ([4150d97](https://github.com/googleapis/go-genai/commit/4150d9759ab3cd35a8bea3040991c1e906f8e787))

## [1.45.0](https://github.com/googleapis/go-genai/compare/v1.44.0...v1.45.0) (2026-02-04)


### Features

* Update data types from discovery doc. ([10f6ed9](https://github.com/googleapis/go-genai/commit/10f6ed99013387f1435f9d0fbdd89bc15581485f))

## [1.44.0](https://github.com/googleapis/go-genai/compare/v1.43.0...v1.44.0) (2026-01-30)


### Features

* Support distillation tuning ([cf4c39c](https://github.com/googleapis/go-genai/commit/cf4c39c1a88af7c00ebe92a4a04a9def3de7939d))
* Support OSS Tuning in GenAI SDK ([9621775](https://github.com/googleapis/go-genai/commit/962177563194d9ff8021bec3afff45e4b9ec7ebc))


### Bug Fixes

* Add metadata in batch inlined response ([f8e6b9b](https://github.com/googleapis/go-genai/commit/f8e6b9b9fa6251f798ae581448bd291dfe5398ec))

## [1.43.0](https://github.com/googleapis/go-genai/compare/v1.42.0...v1.43.0) (2026-01-18)


### Features

* Add ModelArmorConfig support for prompt and response sanitization via the Model Armor service ([580063f](https://github.com/googleapis/go-genai/commit/580063fe5dce4998d0069aec02f6c6d0c01b6016))
* Update data types from discovery doc. ([6842c63](https://github.com/googleapis/go-genai/commit/6842c631b5502558ffec2b8faa6474b74f5374d0))
* Update data types from discovery doc. ([8065f85](https://github.com/googleapis/go-genai/commit/8065f85e4e6f2c5210a20e4202032cc8a557babc))

## [1.42.0](https://github.com/googleapis/go-genai/compare/v1.41.1...v1.42.0) (2026-01-15)


### Features

* voice activity support ([b7bfe9b](https://github.com/googleapis/go-genai/commit/b7bfe9b2253f1a948c9128ff73ef45af7819ba9d))

## [1.41.1](https://github.com/googleapis/go-genai/compare/v1.41.0...v1.41.1) (2026-01-14)


### Bug Fixes

* Correct json tag typo in EntityLabel ([096bff8](https://github.com/googleapis/go-genai/commit/096bff81a61d2d273b2e6a205a614000a06fc26d))

## [1.41.0](https://github.com/googleapis/go-genai/compare/v1.40.0...v1.41.0) (2026-01-07)


### Features

* [Python] add RegisterFiles so gcs files can be used with genai. ([3062834](https://github.com/googleapis/go-genai/commit/3062834b6504280e64de51f82a396d51043292f6))
* Add gemini-3-pro-preview support for local tokenizer ([1eeac81](https://github.com/googleapis/go-genai/commit/1eeac81c22c509d77a632e52c7cd1b9eec5eec10))
* Add PersonGeneration to ImageConfig for Vertex Gempix ([062e7e1](https://github.com/googleapis/go-genai/commit/062e7e151b9816aeebc36c630ec12438a3eb9cfd))

## [1.40.0](https://github.com/googleapis/go-genai/compare/v1.39.0...v1.40.0) (2025-12-16)


### Features

* Add minimal and medium thinking levels. ([a7c0ed8](https://github.com/googleapis/go-genai/commit/a7c0ed8b1fcade06ffcc62a91344e36e56d17f37))
* Add ultra high resolution to the media resolution in Parts. ([47b89eb](https://github.com/googleapis/go-genai/commit/47b89ebb0cb9531bd440afb15df14a9668142f72))
* ephemeral token support in go ([7515bbe](https://github.com/googleapis/go-genai/commit/7515bbef37d86ac0d695c11d81c31a42cb330e7e))
* support multi speaker for Vertex AI ([457541d](https://github.com/googleapis/go-genai/commit/457541d1839e37fe7bd588462ecb42b670a1ec1c))

## [1.39.0](https://github.com/googleapis/go-genai/compare/v1.38.0...v1.39.0) (2025-12-10)


### Features

* Add enableEnhancedCivicAnswers feature in GenerateContentConfig ([dd25df7](https://github.com/googleapis/go-genai/commit/dd25df730c6060e39ab3ba9d7749d8fc75d6c8b7))
* Add IMAGE_RECITATION and IMAGE_OTHER enum values to FinishReason ([33aa4f2](https://github.com/googleapis/go-genai/commit/33aa4f28afcea135e8f8947341ff334afc85fe70))
* Add voice activity detection signal. ([326059c](https://github.com/googleapis/go-genai/commit/326059c108bdc628bf35d75837b1bb23b1b1fa99))

## [1.38.0](https://github.com/googleapis/go-genai/compare/v1.37.0...v1.38.0) (2025-12-08)


### Features

* Support ReplicatedVoiceConfig ([fd19393](https://github.com/googleapis/go-genai/commit/fd1939328de25eda7452630c06ef57f2bcc0d5f9))

## [1.37.0](https://github.com/googleapis/go-genai/compare/v1.36.0...v1.37.0) (2025-12-02)


### Features

* Add empty response for tunings.cancel() ([ace417d](https://github.com/googleapis/go-genai/commit/ace417dc92ed4a84979160cc8eacf2e0ed72f98e))

## [1.36.0](https://github.com/googleapis/go-genai/compare/v1.35.0...v1.36.0) (2025-11-17)


### Features

* add display name to FunctionResponseBlob ([66bd0fb](https://github.com/googleapis/go-genai/commit/66bd0fbf3504ac9f8b01d173a14110af0fabe41d))
* add display name to FunctionResponseFileData ([f470cff](https://github.com/googleapis/go-genai/commit/f470cff5a212cbc0fbda00177b2bfb69c56eb6b1))
* Add generate_content_config.thinking_level ([93b2586](https://github.com/googleapis/go-genai/commit/93b2586dfb392b5f51f36cbbc9d2ac96458c2959))
* Add image output options to ImageConfig for Vertex ([c4b28c3](https://github.com/googleapis/go-genai/commit/c4b28c34141198d96941cad4837edce76bbe764c))
* Add part.media_resolution ([93b2586](https://github.com/googleapis/go-genai/commit/93b2586dfb392b5f51f36cbbc9d2ac96458c2959))
* support Function call argument streaming for all languages ([dd5ec01](https://github.com/googleapis/go-genai/commit/dd5ec01f2186ad777be914f0b33a9e87a2e947d4))

## [1.35.0](https://github.com/googleapis/go-genai/compare/v1.34.0...v1.35.0) (2025-11-12)


### Features

* Add `ToGenerationConfig` method to `GenerateContentConfig`. fixes [#506](https://github.com/googleapis/go-genai/issues/506) ([bd53df4](https://github.com/googleapis/go-genai/commit/bd53df47bb10e8d52dbf5cc200b9c33222947eb5))


### Bug Fixes

* Add missing fields to the model types ([9e1b329](https://github.com/googleapis/go-genai/commit/9e1b3290976f0bf0e640c49a50a8f864c93e7da4))
* Fix base_steps parameter for recontext_image ([4f90d03](https://github.com/googleapis/go-genai/commit/4f90d03d5790fae86f3ea18601f7ce98aba99568))
* Fix models.list() filter parameter ([f5859fd](https://github.com/googleapis/go-genai/commit/f5859fd6b36a6b5c9a6deb28434591ee58b60230))

## [1.34.0](https://github.com/googleapis/go-genai/compare/v1.33.0...v1.34.0) (2025-11-05)


### Features

* Add `UseDefaultCredentials` method to `ClientConfig`. ([d15baab](https://github.com/googleapis/go-genai/commit/d15baab4f79e01d396fa4e8f707cdcd19f6ce404))
* add complete stats to BatchJob ([0df87d7](https://github.com/googleapis/go-genai/commit/0df87d71edaa743f42665e403dd834af81fa6d33))
* Add FileSearch tool and associated FileSearchStore management APIs ([8ce95c8](https://github.com/googleapis/go-genai/commit/8ce95c8eb5fc5b49a74b604729cc967596184850))
* Add FileSearch tool and associated FileSearchStore management APIs ([3e0a744](https://github.com/googleapis/go-genai/commit/3e0a74410171fc42b6d94e9ca18ca2c8a2b8d6c4))
* Add image_size to ImageConfig (Early Access Program) ([027f29c](https://github.com/googleapis/go-genai/commit/027f29ce42fc22bf4861a10a5b5f35ad6ebb77e4))
* Added phish filtering feature. ([eb849d1](https://github.com/googleapis/go-genai/commit/eb849d1e13be374f6751fe7374ab0a58a547781c))


### Bug Fixes

* prevent nil pointer dereference panic in newAPIError function ([251f7bf](https://github.com/googleapis/go-genai/commit/251f7bf00c59b4cbbf74ad35232dce998341df75))
* prevent nil pointer dereference panic in newAPIError function ([#468](https://github.com/googleapis/go-genai/issues/468)) ([ba15429](https://github.com/googleapis/go-genai/commit/ba15429569bdb82ecb78618afa8be2f05ed6c4e2))

## [1.33.0](https://github.com/googleapis/go-genai/compare/v1.32.0...v1.33.0) (2025-10-29)


### Features

* Add safety_filter_level and person_generation for Imagen upscaling ([3d7b57a](https://github.com/googleapis/go-genai/commit/3d7b57a389322c52e2bd8901a6fae0dbdd2c3720))
* Add support for preference optimization tuning in the SDK. ([a6db7fe](https://github.com/googleapis/go-genai/commit/a6db7fe8233501c8457ec7fb8d6b3a132bfb7944))
* Pass file name to the backend when uploading with a file path ([6b5e4f4](https://github.com/googleapis/go-genai/commit/6b5e4f4939a8b13da27fab65731a3423191e2790))
* support default global location when not using api key with vertexai backend ([44f01d6](https://github.com/googleapis/go-genai/commit/44f01d636bff7310bdcfcc5200118fe6ea4a5e2b))

## [1.32.0](https://github.com/googleapis/go-genai/compare/v1.31.0...v1.32.0) (2025-10-21)


### Features

* Add enable_enhanced_civic_answers in GenerationConfig ([c876512](https://github.com/googleapis/go-genai/commit/c87651298d18a77e27f8daba2db0e19074408781))
* support jailbreak in HarmCategory and BlockedReason ([65e1efc](https://github.com/googleapis/go-genai/commit/65e1efc38ca1e5f958be90eac83668b357187187))


### Bug Fixes

* change back SingleEmbedContentResponse.TokenCount to int64 due to mistake in [#557](https://github.com/googleapis/go-genai/issues/557) ([e05f15d](https://github.com/googleapis/go-genai/commit/e05f15d3df064f9ec0e79ab3b6b08fea540e7803))
* do not append project and client for Vertex AI when using API key ([c27f13a](https://github.com/googleapis/go-genai/commit/c27f13a44e51f770bc363b6c8922f0a1cb29ae42)), closes [#564](https://github.com/googleapis/go-genai/issues/564)

## [1.31.0](https://github.com/googleapis/go-genai/compare/v1.30.0...v1.31.0) (2025-10-15)


### Features

* support CreateEmebddings in batches.go ([a488476](https://github.com/googleapis/go-genai/commit/a48847605327c60c49b8276d197bb1dca443730e))
* Support video extension for Veo on Gemini Developer API ([14ecba9](https://github.com/googleapis/go-genai/commit/14ecba9e08e02eed6c72d8bed3b381eb8f31d0fd))

## [1.30.0](https://github.com/googleapis/go-genai/compare/v1.29.0...v1.30.0) (2025-10-10)


### Features

* Enable Google Maps tool for Genai. ([9aba5c7](https://github.com/googleapis/go-genai/commit/9aba5c7ee99884d0ae5a8b3d94f1abc128523a37))
* Support enableWidget feature in GoogleMaps ([cd1e6b7](https://github.com/googleapis/go-genai/commit/cd1e6b7b1b3d79b28a580f787abc5f296522f313))
* Support Gemini batch inline request's metadata and add test coverage to safety setting ([f12f353](https://github.com/googleapis/go-genai/commit/f12f3530050f06d34fa280b8827f9c7514f3c98f))

## [1.29.0](https://github.com/googleapis/go-genai/compare/v1.28.0...v1.29.0) (2025-10-08)


### Features

* Add labels field to Imagen configs ([d01fe5e](https://github.com/googleapis/go-genai/commit/d01fe5e157f9c27a61464ed99c816b73d64d6dbb))
* Add utility methods for creating `FunctionResponsePart` and creating FunctionResponse `Part` with `FunctionResponseParts` ([10de2ab](https://github.com/googleapis/go-genai/commit/10de2ab112ae5d3c67f0591a5c58412f40206372))
* Enable Ingredients to Video and Advanced Controls for Veo on Gemini Developer API (Early Access Program) ([3165554](https://github.com/googleapis/go-genai/commit/31655546a8254f2c0571da7a9d15d00b6111ad6a))

## [1.28.0](https://github.com/googleapis/go-genai/compare/v1.27.0...v1.28.0) (2025-10-02)


### Features

* Add `NO_IMAGE` enum value to `FinishReason` ([4f65f45](https://github.com/googleapis/go-genai/commit/4f65f457b41b112312dc3fa771ee61ef4692b402))
* Add thinking_config for live ([54152b2](https://github.com/googleapis/go-genai/commit/54152b2e6509b1a6fea20df829e366e388bfe6fc))

## [1.27.0](https://github.com/googleapis/go-genai/compare/v1.26.0...v1.27.0) (2025-10-01)


### Features

* Add `ImageConfig` to `GenerateContentConfig` ([55399fd](https://github.com/googleapis/go-genai/commit/55399fdab38bdf88857ed1a4bc50822780f42520))
* rename ComputerUse tool (early access) ([d976966](https://github.com/googleapis/go-genai/commit/d9769665237fb266eb0063af26124e7d76557ca4))


### Bug Fixes

* fix system_instruction mapping issue in batches module ([c862a6d](https://github.com/googleapis/go-genai/commit/c862a6de5307a363478bee210d73c6f1e3199d8b))

## [1.26.0](https://github.com/googleapis/go-genai/compare/v1.25.0...v1.26.0) (2025-09-25)


### Features

* Add FunctionResponsePart & ToolComputerUse.excludedPredefinedFunctions ([8b97cef](https://github.com/googleapis/go-genai/commit/8b97cefe4683fdba3be021d8c7025b46c5327a42))
* Support Imagen 4 Ingredients on Vertex ([a9ab364](https://github.com/googleapis/go-genai/commit/a9ab364b51b120b119abf62f6bb6aecc32684775))


### Bug Fixes

* Expose `JOB_STATE_RUNNING` and `JOB_STATE_EXPIRED` for Gemini Batches states ([26e0182](https://github.com/googleapis/go-genai/commit/26e01826137144aaaa0035600eb28c6b194283ab))
* fix systemInstruction mapping issue in batch generate content. fixes [#505](https://github.com/googleapis/go-genai/issues/505) ([3997ea2](https://github.com/googleapis/go-genai/commit/3997ea21bb6447848980037eb3d269f4bf7eacda))
* initialization of `pre_tuned_model_checkpoint_id` from tuning config. ([b093bcf](https://github.com/googleapis/go-genai/commit/b093bcf876c8e84fe5675373e4485554aa3015a7))
* Prevent adding `,string` JSON tag for `int64`/`uint64` fields in `Schema` class. fixes [#511](https://github.com/googleapis/go-genai/issues/511) [#481](https://github.com/googleapis/go-genai/issues/481) ([3423dd3](https://github.com/googleapis/go-genai/commit/3423dd359db32d55c7e91008f7a5475cc1eb790c))

## [1.25.0](https://github.com/googleapis/go-genai/compare/v1.24.0...v1.25.0) (2025-09-16)


### Features

* Add 'turn_complete_reason' and 'waiting_for_input' fields. ([2d48288](https://github.com/googleapis/go-genai/commit/2d48288d405b16474011e018cbb10096c5957c93))
* Add `VideoGenerationMaskMode` enum for Veo 2 Editing ([65f9058](https://github.com/googleapis/go-genai/commit/65f9058b728748bfe8b83d0178300249c3700d10))
* local tokenizer for text ([ee46a52](https://github.com/googleapis/go-genai/commit/ee46a52de5e0e8678f0be3269de4cf6a60c90696))

## [1.24.0](https://github.com/googleapis/go-genai/compare/v1.23.0...v1.24.0) (2025-09-09)


### Features

* [Python] Implement async embedding batches for MLDev. ([f32fb26](https://github.com/googleapis/go-genai/commit/f32fb26a125e0df212c1a27615de4899f4ba791a))
* Add labels to create tuning job config ([c13a2a5](https://github.com/googleapis/go-genai/commit/c13a2a5f68d867122d2d7e3a7f2b58784c9df20f))
* generate the function_call class's converters ([995a3ac](https://github.com/googleapis/go-genai/commit/995a3acc0a7bd9bdf3048fb332f23c21a36a2b62))
* Support Veo 2 Editing on Vertex ([7fd6940](https://github.com/googleapis/go-genai/commit/7fd694074b5488b0eb2d5d0cf2f4e0d75de16533))


### Bug Fixes

* Enable `id` field in `FunctionCall` for Vertex AI. ([a3f3c2b](https://github.com/googleapis/go-genai/commit/a3f3c2b37ef065b43cb4ab27f7b60c0c89a8684c))

## [1.23.0](https://github.com/googleapis/go-genai/compare/v1.22.0...v1.23.0) (2025-09-02)


### Features

* Add resolution field for Gemini Developer API Veo 3 generation ([b6a989c](https://github.com/googleapis/go-genai/commit/b6a989cdcad0a3f006a1753f6bac5e91df2914ba))

## [1.22.0](https://github.com/googleapis/go-genai/compare/v1.21.0...v1.22.0) (2025-08-27)


### Features

* add `sdkHttpResponse.headers` to *Delete responses. ([ac0e763](https://github.com/googleapis/go-genai/commit/ac0e7632e5760378d0751cc8c9224bbd6a1bad0c))
* Add add_watermark field for recontext_image (Virtual Try-On, Product Recontext) ([07b6f57](https://github.com/googleapis/go-genai/commit/07b6f573b2941fd22e5d68df09610f0d910b3463))
* Add output_gcs_uri to Imagen upscale_image ([c08d9f3](https://github.com/googleapis/go-genai/commit/c08d9f35c3dce6da9854842926aeec61593ff02a))
* Add VALIDATED mode into FunctionCallingConfigMode ([c282e79](https://github.com/googleapis/go-genai/commit/c282e79bed1ac0fd60facbffe3dde74b8d240a01))
* Add VideoGenerationReferenceType enum for generate_videos ([635b825](https://github.com/googleapis/go-genai/commit/635b825bedd2bbe5d2e84eca78e0d3b08eecdc31))
* refactor Go SDK to use pointers for optional parameters ([3ff328a](https://github.com/googleapis/go-genai/commit/3ff328ac19ce7b4429ce6d75f2fe2d1ffaf37a37))
* support tunings.cancel in the genai SDK for Python, Java, JS, and Go ([8c46fd2](https://github.com/googleapis/go-genai/commit/8c46fd26e1985f15d510ed0f4d4cefcdd2110af7))

## [1.21.0](https://github.com/googleapis/go-genai/compare/v1.20.0...v1.21.0) (2025-08-18)


### Features

* Support Imagen image segmentation on Vertex ([2a38843](https://github.com/googleapis/go-genai/commit/2a388434bc2bd9d564479e1d8db6eb4ffdadcb68))
* Support Veo 2 Reference Images to Video Generation on Vertex ([9894324](https://github.com/googleapis/go-genai/commit/9894324c30a4a73614f1c4ed9ce9ebf67eb7a5a9))


### Bug Fixes

* Add a missing resp.Body.Close() to deserializeStreamResponse. ([bf3fb3f](https://github.com/googleapis/go-genai/commit/bf3fb3f37172fbff4d4045ebe781d8f03f7fb23a))

## [1.20.0](https://github.com/googleapis/go-genai/compare/v1.19.0...v1.20.0) (2025-08-13)


### Features

* enable continuous fine-tuning on a pre-tuned model in the SDK. ([1f20493](https://github.com/googleapis/go-genai/commit/1f204939b8d68d8f5c9d739ade69716a03be28f1))
* support document name in grounding metadata ([f673e20](https://github.com/googleapis/go-genai/commit/f673e200ce5663de3f62b832a4bafc756494da75))
* Support exclude_domains in Google Search and Enterprise Web Search ([2547ad1](https://github.com/googleapis/go-genai/commit/2547ad1a4f07ac1111b14a6352a64bc4d7bea5a5))

## [1.19.0](https://github.com/googleapis/go-genai/compare/v1.18.0...v1.19.0) (2025-08-06)


### Features

* Add image_size field for Gemini Developer API Imagen 4 generation ([3ccd2b0](https://github.com/googleapis/go-genai/commit/3ccd2b086ecc761dd0492bae030f9ce7501e17c1))
* Add parts length check ([#444](https://github.com/googleapis/go-genai/issues/444)) ([a4896f3](https://github.com/googleapis/go-genai/commit/a4896f37c756e9cf9b69a9ed38db00bf36e28f0d))
* allow methods in batch to return headers in sdk_http_response by default ([3a1d6d8](https://github.com/googleapis/go-genai/commit/3a1d6d83a10b53fc44c7cc06e1d3b0426a437c98))
* enable responseId for Gemini Developer API ([9845a91](https://github.com/googleapis/go-genai/commit/9845a919d768d7ced4857518b3155426105c5d85))
* support curated history in the Go chats module. ([2b99b83](https://github.com/googleapis/go-genai/commit/2b99b83451ed6e622e985baf68461c3ad03e793e))
* Support image recontext on Vertex ([fc7ee78](https://github.com/googleapis/go-genai/commit/fc7ee78fcae0f196deeaf8ccac4592905112c60c))
* Support new enum types for UrlRetrievalStatus ([ef63a73](https://github.com/googleapis/go-genai/commit/ef63a73505062e763df5f33559c6b0ba160e22ca))

## [1.18.0](https://github.com/googleapis/go-genai/compare/v1.17.0...v1.18.0) (2025-07-30)


### Features

* support response headers in Go for all methods. ([4865b93](https://github.com/googleapis/go-genai/commit/4865b9366e72f1ee5f124346bd15198d3582e7f1))

## [1.17.0](https://github.com/googleapis/go-genai/compare/v1.16.0...v1.17.0) (2025-07-17)


### Features

* Add generateVideosFromSource in Go, refactor private generateVideos ([cd625ec](https://github.com/googleapis/go-genai/commit/cd625ec384e059ae5c09587421ad089a76d9ab8b))
* Add image_size field for Vertex Imagen 4 generation ([5458a1c](https://github.com/googleapis/go-genai/commit/5458a1c3a422c1575328c7bb8aed9f8241dcac34))
* Support HTTPOptions.ExtraBody ([bce0d4a](https://github.com/googleapis/go-genai/commit/bce0d4a53e78a61c8598d9004a9f228c204e58e4))

## [1.16.0](https://github.com/googleapis/go-genai/compare/v1.15.0...v1.16.0) (2025-07-16)


### Features

* Add `addWatermark` parameter to the edit image configuration. ([d7e2847](https://github.com/googleapis/go-genai/commit/d7e284753de40e21ea6df0a5e821f9973d8aa3e3))
* add Tuning support for Go ([0c90d6c](https://github.com/googleapis/go-genai/commit/0c90d6c7471f9634ef539a98a6dc1f15cf67446b))
* Migrate Go file.create method to use the sdk_http_response field and remove http_headers ([082b7c7](https://github.com/googleapis/go-genai/commit/082b7c7ad7a9f447e0ea07e8cb8f949fef578572))


### Bug Fixes

* **live:** Enhance security by moving api key from query parameters to header ([7cc7b7c](https://github.com/googleapis/go-genai/commit/7cc7b7c722d554796bec9f8bbd1af374be0b4892))

## [1.15.0](https://github.com/googleapis/go-genai/compare/v1.14.0...v1.15.0) (2025-07-09)


### Features

* Add new languages for Imagen 4 prompt language ([afeabc2](https://github.com/googleapis/go-genai/commit/afeabc2afbac49b25eb7cd157192445da5f545d3))
* make credentials optional when providing HTTPClient ([8b63004](https://github.com/googleapis/go-genai/commit/8b630040d5ca97f6bcf627f28a0628a944ae8432))

## [1.14.0](https://github.com/googleapis/go-genai/compare/v1.13.0...v1.14.0) (2025-07-01)


### Features

* Enable HttpOptions timeout ([1ca3aaf](https://github.com/googleapis/go-genai/commit/1ca3aaf8f4cfa786226eba1dbfb361dda574f2f7))
* Support Batches delete ([b2cf7bb](https://github.com/googleapis/go-genai/commit/b2cf7bb6b1f091fb18e4a8bb40e07480e7d448a4))
* Support different media input in Vertex Live API ([b6650b5](https://github.com/googleapis/go-genai/commit/b6650b51d88028cab0eb674028aef130185ae560))

## [1.13.0](https://github.com/googleapis/go-genai/compare/v1.12.0...v1.13.0) (2025-06-25)


### Features

* Add compressionQuality enum for generate_videos ([061e567](https://github.com/googleapis/go-genai/commit/061e5674e1744d3bdaade62d3fb8f94c65e77c4c))
* Add enhance_input_image and image_preservation_factor fields for upscale_image ([6b9e07f](https://github.com/googleapis/go-genai/commit/6b9e07fee4864b54690e84ee9b67cf8a588d89d9))
* Batches support in Go ([dcae33f](https://github.com/googleapis/go-genai/commit/dcae33f83abfbad8b3cd14930ca0514a61a217d8))
* expose the responseJsonSchema in GenerateContentConfig ([611adde](https://github.com/googleapis/go-genai/commit/611adde2f1989e66b6e5d029f7d077e7630ada71))

## [1.12.0](https://github.com/googleapis/go-genai/compare/v1.11.1...v1.12.0) (2025-06-18)


### Features

* enable json schema for controlled output and function declaration. ([f2abe6b](https://github.com/googleapis/go-genai/commit/f2abe6b739b57ef46b1f7b459b9ce1fed3a6956c))

## [1.11.1](https://github.com/googleapis/go-genai/compare/v1.11.0...v1.11.1) (2025-06-13)


### Bug Fixes

* Exclude video field from GenerateVideos to restore backwards compatibility in Go SDK ([a868b8b](https://github.com/googleapis/go-genai/commit/a868b8b1f989a486ff183e48ecc76891def14c41))
* Fix precedence when reading from environment variables api key and project for Go SDK ([46bf8cc](https://github.com/googleapis/go-genai/commit/46bf8ccc2b0f88ee50f352ef04bb8ae73f0feb43))

## [1.11.0](https://github.com/googleapis/go-genai/compare/v1.10.0...v1.11.0) (2025-06-11)


### Features

* Add datastore_spec field for VertexAISearch ([489ee80](https://github.com/googleapis/go-genai/commit/489ee809e71b5dbbf49757e2d68c29f0295445a1))
* Add support for Veo frame interpolation and video extension ([eae965b](https://github.com/googleapis/go-genai/commit/eae965bbdb60befee32b7496ca01aa172890cb9b))
* RAG - Introducing context storing for Gemini Live API. ([b752aa0](https://github.com/googleapis/go-genai/commit/b752aa0656bb72ccf9d2eeb4d12585d0ec04743e))
* Support API keys for VertexAI mode for Go SDK ([3b4aadf](https://github.com/googleapis/go-genai/commit/3b4aadf46f3ac4b075bec608657e81f2c8163069))


### Bug Fixes

* handle structured error in the stream chunk. fixes [#355](https://github.com/googleapis/go-genai/issues/355) ([fb361ac](https://github.com/googleapis/go-genai/commit/fb361acd2705a19a113aa192aee9205d3542cdd3))

## [1.10.0](https://github.com/googleapis/go-genai/compare/v1.9.0...v1.10.0) (2025-06-05)


### Features

* add extras request provider to the HTTP options ([7a00367](https://github.com/googleapis/go-genai/commit/7a00367377e00d41a6b599fbd0a9cc08ba03ab81))


### Bug Fixes

* Merge ExtrasRequestProvider for client level config and function level config ([f63de00](https://github.com/googleapis/go-genai/commit/f63de00a51227fb5aec4bd72a738ae60b94d499e))

## [1.9.0](https://github.com/googleapis/go-genai/compare/v1.8.0...v1.9.0) (2025-06-04)


### Features

* Add enhance_prompt field for Gemini Developer API generate_videos ([04c9207](https://github.com/googleapis/go-genai/commit/04c92074d3c12b4cd2b00834086eb33c57c11a31))
* Enable url_context for Vertex ([6ca5b9c](https://github.com/googleapis/go-genai/commit/6ca5b9c20f41d54e5ef0856f008cd15b065ca5d0))
* **go:** Support `GEMINI_API_KEY` as environment variable for setting API key. ([83ba1e0](https://github.com/googleapis/go-genai/commit/83ba1e01082521c6160f8f7f87eefe32e66a6c2b))

## [1.8.0](https://github.com/googleapis/go-genai/compare/v1.7.0...v1.8.0) (2025-05-30)


### Features

* Adding `thought_signature` field to the `Part` to store the signature for thoughts. ([080fd90](https://github.com/googleapis/go-genai/commit/080fd90ce79acdf77cb74f803f721e643e711740))
* include UNEXPECTED_TOOL_CALL enum value to FinishReason for Vertex AI APIs. ([03f0ea1](https://github.com/googleapis/go-genai/commit/03f0ea1c1bd94b8848c0e4abfa1c63a765ba7673))


### Bug Fixes

* Rename LiveEphemeralParameters to LiveConnectConstraints. ([e7c5ee7](https://github.com/googleapis/go-genai/commit/e7c5ee71b6d028ded68826c0b3358372a8d38f13))

## [1.7.0](https://github.com/googleapis/go-genai/compare/v1.6.0...v1.7.0) (2025-05-28)


### Features

* Add generate_audio field for private testing of video generation ([d48d6f3](https://github.com/googleapis/go-genai/commit/d48d6f354e8d914c65f65244d87046d59167db3b))
* support new fields in FileData, GenerationConfig, GroundingChunkRetrievedContext, RetrievalConfig, Schema, TuningJob, VertexAISearch, ([9331c82](https://github.com/googleapis/go-genai/commit/9331c8285a9be325226ab32307db2e3fdb007652))


### Bug Fixes

* use correct mimetype for image content ([#301](https://github.com/googleapis/go-genai/issues/301)) ([ddc69b8](https://github.com/googleapis/go-genai/commit/ddc69b8a3d6eca1130964dcf838764b8c8de41d7))

## [1.6.0](https://github.com/googleapis/go-genai/compare/v1.5.0...v1.6.0) (2025-05-19)


### Features

* add `time range filter` to Google Search Tool ([02bec9d](https://github.com/googleapis/go-genai/commit/02bec9d69fe6d28848c09f23897c4b5149825250))
* Add basic support for async function calling. ([514cf37](https://github.com/googleapis/go-genai/commit/514cf37e73f27dd3d70745222f50a100b368c5dc))
* add live proactivity_audio and enable_affective_dialog ([a72f5ce](https://github.com/googleapis/go-genai/commit/a72f5ce369b8dfa77845dd818d2146c584919307))
* add multi-speaker voice config ([aae87a9](https://github.com/googleapis/go-genai/commit/aae87a9c251c20c0739dae9c2a5423847d1a9cd1))
* Add support for lat/long in search. ([0e2ba95](https://github.com/googleapis/go-genai/commit/0e2ba95e523e6733709146e6e6b776f8ea8011d3))
* Add Video FPS, and enable start/end_offset for MLDev ([fa403ac](https://github.com/googleapis/go-genai/commit/fa403ac4ff1908e71cfe695934d150c023fdb773))
* support customer-managed encryption key in cached content ([a8a6dc2](https://github.com/googleapis/go-genai/commit/a8a6dc2894ba4d037f1734a97dc457b54b68487e))
* Support Url Context Retrieval tool ([7bf9acc](https://github.com/googleapis/go-genai/commit/7bf9acc8bed7d38d8780e6a1c7548c90008c463e))


### Bug Fixes

* fix SendMessageStream when iterator callback returns false. fixes [#310](https://github.com/googleapis/go-genai/issues/310) ([08cf7d9](https://github.com/googleapis/go-genai/commit/08cf7d9c5bc9f50e2feb4093c5952b6668d7e36b))

## [1.5.0](https://github.com/googleapis/go-genai/compare/v1.4.0...v1.5.0) (2025-05-13)


### Features

* Add Send and SendStream to chats module. Related to [#295](https://github.com/googleapis/go-genai/issues/295) ([f2f8041](https://github.com/googleapis/go-genai/commit/f2f80410f7924eca0d59ee8ff02ff7f8ffae22ce))
* support display_name for Blob class when calling Vertex AI ([10b5438](https://github.com/googleapis/go-genai/commit/10b5438d997ea8ae05931dedf7037f369c149576))
* Support tuning checkpoints ([c65e033](https://github.com/googleapis/go-genai/commit/c65e0335471317f2fec29fe2161ef57ca31e3abe))

## [1.4.0](https://github.com/googleapis/go-genai/compare/v1.3.0...v1.4.0) (2025-05-08)


### Features

* Add `Tool.enterprise_web_search` field ([452d379](https://github.com/googleapis/go-genai/commit/452d379e1a9cffb722c34e038af89539855fdc69))
* Add support for Grounding with Google Maps ([76c6472](https://github.com/googleapis/go-genai/commit/76c6472eac47f6c5b1f8875d3d9d1fd198db3e77))
* enable input transcription for Gemini API. ([f1ccf67](https://github.com/googleapis/go-genai/commit/f1ccf67133ee0c929661e821790768775db3f9fc))


### Bug Fixes

* add retry logic for missing X-Goog-Upload-Status header for golang ([1a25f15](https://github.com/googleapis/go-genai/commit/1a25f159557ce461598483b144ab9adfe4c85a95))

## [1.3.0](https://github.com/googleapis/go-genai/compare/v1.2.0...v1.3.0) (2025-04-30)


### Features

* add models.delete and models.update to manage tuned models ([6f5bbed](https://github.com/googleapis/go-genai/commit/6f5bbed7ab1514d246befe4e8aa16c4244678b25))
* add NewPartFromFile for File type convenience. Related to [#281](https://github.com/googleapis/go-genai/issues/281) ([2ac0429](https://github.com/googleapis/go-genai/commit/2ac0429c634edca2396c272f27f013b4a960529a))
* add support for live grounding metadata ([9ce2ed9](https://github.com/googleapis/go-genai/commit/9ce2ed91ffbe781b513487ca29b86d95497e8f06))
* make min_property, max_property, min_length, max_length, example, patter fields available for Schema class when calling Gemini API ([b487724](https://github.com/googleapis/go-genai/commit/b48772435585584c9974b923d071fe567cd00366))
* Populate X-Server-Timeout header when a request timeout is set. ([8f446a0](https://github.com/googleapis/go-genai/commit/8f446a0a0c5ffe74ec00e32b0667d830596db49d))
* Support setting the default base URL in clients via SetDefaultBaseURLs() ([f465a20](https://github.com/googleapis/go-genai/commit/f465a2088c0d4a104f4edbd7197b64f1097fc1b8))


### Bug Fixes

* do not raise error for `default` field in Schema for Gemini API calls ([ec31e4b](https://github.com/googleapis/go-genai/commit/ec31e4b92afca2f167456ef5e8f775cfad198b8d))
* do not remove content parts with `Text` unset. ([b967057](https://github.com/googleapis/go-genai/commit/b967057d68ad8cd5385aa19b65b0648646cb8c00))
* **files:** deep copy config struct before modifying it. ([a6b0fd6](https://github.com/googleapis/go-genai/commit/a6b0fd6c47cfda6ce28d01a8119ae6f38e2214f4))

## [1.2.0](https://github.com/googleapis/go-genai/compare/v1.1.0...v1.2.0) (2025-04-23)


### Features

* add additional realtime input fields ([1190539](https://github.com/googleapis/go-genai/commit/11905391266e0ee01e7eff4ee68304bcc3654f36))
* Expose transcription configurations for audio in TS, and move generationConfig to the top level LiveConnectConfig ([ead7e49](https://github.com/googleapis/go-genai/commit/ead7e49e4e710a31bd87cf18d0f7d5925bae8662))
* support `default` field in Schema when users call Gemini API ([643eb80](https://github.com/googleapis/go-genai/commit/643eb801b141ab27e05d77678f179c9a1dc5407c))

## [1.1.0](https://github.com/googleapis/go-genai/compare/v1.0.0...v1.1.0) (2025-04-16)


### Features

* Add converters to support continuous sessions with a sliding window ([97bbba4](https://github.com/googleapis/go-genai/commit/97bbba4655ec1b754532c10d464ed037e7312158))
* add support for model_selection_config to GenerateContentConfig ([a44a6c7](https://github.com/googleapis/go-genai/commit/a44a6c78bb4e2bc614bcf526930338e1fd6b84a7))
* Add types for configurable speech detection ([658c17a](https://github.com/googleapis/go-genai/commit/658c17a226256decb52c59e595d360867cf987ea))
* Support audio transcription in Vertex Live API ([f09dfab](https://github.com/googleapis/go-genai/commit/f09dfab7de4a8bb4abf52302b0df3749a7e043a6))
* Support RealtimeInputConfig, and language_code in SpeechConfig in python ([f90a0ec](https://github.com/googleapis/go-genai/commit/f90a0ec3bd58759152b5c417c9d725cedbb98fe9))
* Update VertexRagStore ([62d582c](https://github.com/googleapis/go-genai/commit/62d582c9f2a0bd4149f6a87efd3b71886fa969ef))


### Bug Fixes

* **files:** use `io.ReadFull` to read the file. fixes [#237](https://github.com/googleapis/go-genai/issues/237) ([908c783](https://github.com/googleapis/go-genai/commit/908c78371f2c9e16b4bd43286b6e7ed95d02fa8e))
* Fix error "assignment to entry in nil map" of `Files.Upload()` when `config.HTTPOptions` is nil ([#235](https://github.com/googleapis/go-genai/issues/235)) ([05c0c49](https://github.com/googleapis/go-genai/commit/05c0c49512f56dd808408ddf555699af6b164ac3))
* fix MIME type error in UploadFromPath and add unit tests. fixes: [#247](https://github.com/googleapis/go-genai/issues/247) ([f851639](https://github.com/googleapis/go-genai/commit/f851639a7f5bc3d8392a8d2cee2e25ed0d42feda))

## [1.0.0](https://github.com/googleapis/go-genai/compare/v0.7.0...v1.0.0) (2025-04-09)


### ⚠ BREAKING CHANGES

* Support SendClientContent/SendRealtimeInput/SendToolResponse methods in Session struct and remove Send method
* Merge GenerationConfig to LiveConnectConfig. GenerationConfig is removed.
* Change NewContentFrom... functions role param type from string to Role and miscs docstring improvements
* Change some pointer to value type and value to pointer type

### Features

* Add domain to Web GroundingChunk ([183ac49](https://github.com/googleapis/go-genai/commit/183ac49d75bb8a84c95df6aba6b284761509e61e))
* Add generationComplete notification to Live ServerContent ([9a038b9](https://github.com/googleapis/go-genai/commit/9a038b96cc8e979649033a6636387329da443b26))
* Add session resumption to Live module ([4a92461](https://github.com/googleapis/go-genai/commit/4a92461832b60b7a4adf32d99f3a50651c4db50b))
* add session resumption. ([507137b](https://github.com/googleapis/go-genai/commit/507137bcbe76e8e2b4a7372038e3136fb4a36425))
* Add support for Chats streaming in Go SDK ([9ee0523](https://github.com/googleapis/go-genai/commit/9ee0523e4975ddced4b3918ada8bdea4c1a0787f))
* Add thinking_budget to ThinkingConfig for Gemini Thinking Models ([f811ee4](https://github.com/googleapis/go-genai/commit/f811ee48b67db553b7520bc417f366270415d95e))
* Add traffic type to GenerateContentResponseUsageMetadata ([601add2](https://github.com/googleapis/go-genai/commit/601add239ae6722ab84f9bfabe3b0d4a84bf7b42))
* Add types for configurable speech detection ([f4e1b11](https://github.com/googleapis/go-genai/commit/f4e1b118df97866e8b7b47baedde9470cb842ed0))
* Add types to support continuous sessions with a sliding window ([5d4f5d7](https://github.com/googleapis/go-genai/commit/5d4f5d7e5e3ce96f7876fc8a65ef49c5c796a6ad))
* Add UsageMetadata to LiveServerMessage ([4286c6b](https://github.com/googleapis/go-genai/commit/4286c6bf04adee388c4dcdc83c4fe5923558b573))
* expose generation_complete, input/output_transcription & input/output_audio_transcription to SDK for Vertex Live API ([0dbbc82](https://github.com/googleapis/go-genai/commit/0dbbc82a0f03c617d01726468993c58128016dca))
* Merge GenerationConfig to LiveConnectConfig. GenerationConfig is removed. ([65b7c1c](https://github.com/googleapis/go-genai/commit/65b7c1c51e6d954f3c2d61202f6d7b6ba5a8ceb1))
* Remove experimental warnings for generate_videos and operations ([2e4bb0b](https://github.com/googleapis/go-genai/commit/2e4bb0bb12f2eb3a88d4d125ed8bc6c8166e051f))
* Support files delete, get, list, download/ ([8e7b3fd](https://github.com/googleapis/go-genai/commit/8e7b3fd50775ab4ca11484a85a40166066e05f6a))
* Support files upload method ([ce790dd](https://github.com/googleapis/go-genai/commit/ce790ddd9b34c12c913634b890ba5fa01f86c18a))
* support media resolution ([825c81d](https://github.com/googleapis/go-genai/commit/825c81dbcb9eeff54f52052270e1f5d738fab39c))
* Support SendClientContent/SendRealtimeInput/SendToolResponse methods in Session struct and remove Send method ([c8ecaf4](https://github.com/googleapis/go-genai/commit/c8ecaf4ffa2c3f5ca59692af6711651966630729))
* use io.Reader in Upload function and add a new convenience function UploadFromPath. fixes [#222](https://github.com/googleapis/go-genai/issues/222) ([1c064e3](https://github.com/googleapis/go-genai/commit/1c064e3e15c75e987189cb4a65080a4aa087531d))


### Bug Fixes

* Change NewContentFrom... functions role param type from string to Role and miscs docstring improvements ([7810e07](https://github.com/googleapis/go-genai/commit/7810e074299bbd9c38160a995cc6df311a3e9e88))
* Change some pointer to value type and value to pointer type ([0d2ba97](https://github.com/googleapis/go-genai/commit/0d2ba97b813ad51f964306de4399cbdd777105eb))
* fix Add() dead loop ([afa2324](https://github.com/googleapis/go-genai/commit/afa23240ac30a0fafca7877d5034f34a3c187e91))
* Fix failing chat_test ([aebbdaa](https://github.com/googleapis/go-genai/commit/aebbdaa234b2a0552f738c593a46094e6016dedc))

## [0.7.0](https://github.com/googleapis/go-genai/compare/v0.6.0...v0.7.0) (2025-03-31)


### ⚠ BREAKING CHANGES

* Add error return type to Close() function
* consolidate NewUserContentFrom* and NewModelContentFrom* functions into NewContentFrom* to make API simpler
* Support quota project and migrate ClientConfig.Credential from google.Credentials to auth.Credential type.
* Change caches TTL field to duration type.
* rename ClientError and ServerError to APIError. fixes: #159

### Features

* Add Chats module for Go SDK (non-stream only) ([e7f75fd](https://github.com/googleapis/go-genai/commit/e7f75fdd931001e5e3e68c453201ce933a70f064))
* Add engine to VertexAISearch ([cc2ab5d](https://github.com/googleapis/go-genai/commit/cc2ab5dc7013f045d6d7393cc7cbd05988f767da))
* add IMAGE_SAFTY enum value to FinishReason ([cc6081a](https://github.com/googleapis/go-genai/commit/cc6081a7e781fb68a6cbcb89528de85c31c4fb6a))
* add MediaModalities for ModalityTokenCount ([0969afd](https://github.com/googleapis/go-genai/commit/0969afd3854fdec86e001f3412582aa95123286f))
* Add Veo 2 generate_videos support in Go SDK ([5321a25](https://github.com/googleapis/go-genai/commit/5321a25f0134b8b2d45ebdfb73544123044f96c7))
* allow title property to be sent to Gemini API. Gemini API now supports the title property, so it's ok to pass this onto both Vertex and Gemini API. ([8f27aba](https://github.com/googleapis/go-genai/commit/8f27aba6199bfac6205fb7e88883a5c6a1ee017e))
* consolidate NewUserContentFrom* and NewModelContentFrom* functions into NewContentFrom* to make API simpler ([e8608b1](https://github.com/googleapis/go-genai/commit/e8608b19f7bec5cb976095b2d5cdb69886ae6036))
* merge GenerationConfig into LiveConnectConfig ([96232de](https://github.com/googleapis/go-genai/commit/96232de67aa69af0f1e10625961765b13d3dbfc5))
* rename ClientError and ServerError to APIError. fixes: [#159](https://github.com/googleapis/go-genai/issues/159) ([12adbfa](https://github.com/googleapis/go-genai/commit/12adbfae781a1df63a32094895dc0b37baad32da))
* Save prompt safety attributes in dedicated field for generate_images ([eb3cfdc](https://github.com/googleapis/go-genai/commit/eb3cfdc8773b85bae648a90dddaa69435824a58b))
* support new UsageMetadata fields ([3a56c63](https://github.com/googleapis/go-genai/commit/3a56c632f11d703786cb546d4ced3ee7bbf84b39))
* Support quota project and migrate ClientConfig.Credential from google.Credentials to auth.Credential type. ([74c05fb](https://github.com/googleapis/go-genai/commit/74c05fbf68e3c35627d69720d3de733f0d38cbce))


### Bug Fixes

* Add error return type to Close() function ([673a7f7](https://github.com/googleapis/go-genai/commit/673a7f7e61cf4a3377e145d4aed8f54b7d90886f))
* Change caches TTL field to duration type. ([11271b4](https://github.com/googleapis/go-genai/commit/11271b4d888741d5dcaebbe9dea44daface9e198))
* fix list models API url ([036c4d3](https://github.com/googleapis/go-genai/commit/036c4d3e368c1184641e9e089056b57c875e2a10))
* fix response modality in streaming mode. fixes [#163](https://github.com/googleapis/go-genai/issues/163). fixes [#158](https://github.com/googleapis/go-genai/issues/158) ([996dac3](https://github.com/googleapis/go-genai/commit/996dac39f23dff4436dfea3f2badf414f9435338))
* missing zero value bug in setValueByPath. fixes [#196](https://github.com/googleapis/go-genai/issues/196) ([557c6d8](https://github.com/googleapis/go-genai/commit/557c6d8a8de80caf6999fc2ba2be166e140e8880))
* schema transformer logic fix. ([8017092](https://github.com/googleapis/go-genai/commit/8017092b7cfe42a44e5b4b09f4c934ac723618f4))
* use snake_case in embed_content request/response parsing. fixes [#174](https://github.com/googleapis/go-genai/issues/174) ([ba644e1](https://github.com/googleapis/go-genai/commit/ba644e19b03d948487da3b12f843fe32cb3b1851))


### Miscellaneous Chores

* release 0.7.0 ([06523b4](https://github.com/googleapis/go-genai/commit/06523b4d9b90c3dae5dba72331297c5c1d23e28d))

## [0.6.0](https://github.com/googleapis/go-genai/compare/v0.5.0...v0.6.0) (2025-03-19)


### ⚠ BREAKING CHANGES

* support duration type and remove NewPartFromVideoMetadata function
* Change *time.Time type to time.Time.
* remove error from the GenerateContentResponse.Text() return values and add more samples(text embedding, tokens, models)
* change GenerateImageConfig.NumberOfImages to value type. And add clearer error message and docstring to other APIs.
* Remove default role to "user" for GenerateContent and GenerateContentStream.

### Features

* Add base steps to EditImageConfig ([e3c8252](https://github.com/googleapis/go-genai/commit/e3c82523429d43684e898a10991fb86161f5f48f))
* Change *time.Time type to time.Time. ([d554a08](https://github.com/googleapis/go-genai/commit/d554a081fff30d0fec4395ef5d8dd936d81a5477))
* change GenerateImageConfig.NumberOfImages to value type. And add clearer error message and docstring to other APIs. ([a75a9ae](https://github.com/googleapis/go-genai/commit/a75a9ae4d7f782c8894b9c8bc7e9c44f93e71fe6))
* enable union type for Schema when calling Gemini API. ([2edcc55](https://github.com/googleapis/go-genai/commit/2edcc5560a89b76542d77566890911bf1a163795))
* Remove default role to "user" for GenerateContent and GenerateContentStream. ([74d4647](https://github.com/googleapis/go-genai/commit/74d46476678813c1888d89b0112c94f6fa0d3a2e))
* remove error from the GenerateContentResponse.Text() return values and add more samples(text embedding, tokens, models) ([1dc5c1c](https://github.com/googleapis/go-genai/commit/1dc5c1c95acb2f207632eeeeb8fa6d4cbb6a7df4))
* support duration type and remove NewPartFromVideoMetadata function ([b2a13ab](https://github.com/googleapis/go-genai/commit/b2a13ab16cfbd6b167d2541128a75ab059ffc044))
* Support global endpoint in go natively ([a29b806](https://github.com/googleapis/go-genai/commit/a29b806d89dd7ebddb44486c23cd51f79864029d))
* Support returned safety attributes for generate_images ([cc2bf1a](https://github.com/googleapis/go-genai/commit/cc2bf1aa581439b2d674966eed55caa580038a83))


### Bug Fixes

* Make month and day optional for PublicationDate. fixes [#141](https://github.com/googleapis/go-genai/issues/141) ([8a61516](https://github.com/googleapis/go-genai/commit/8a615165d2161f5be0efb0d7bf5f77570166b0b0))
* Remove unsupported parameter negative_prompt from Gemini API generate_images ([be2619d](https://github.com/googleapis/go-genai/commit/be2619d6d2304f680ae8f9b2b669a6799929988b))


### Miscellaneous Chores

* release 0.6.0 ([f636767](https://github.com/googleapis/go-genai/commit/f636767b3fdc4c4a186c9465fdc3cb2d950c158b))

## [0.5.0](https://github.com/googleapis/go-genai/compare/v0.4.0...v0.5.0) (2025-03-06)


### ⚠ BREAKING CHANGES

* change int64, float64 types to int32, unit32, float32 to prevent data loss
* remove ClientConfig.Timeout and add HTTPOptions to ...Config structs

### Features

* Add Headers field into HTTPOption struct ([5ec9ff4](https://github.com/googleapis/go-genai/commit/5ec9ff40ce4e9f3fd4625eab68dfbe5e9d259237))
* Add response_id and create_time to GenerateContentResponse ([f46d996](https://github.com/googleapis/go-genai/commit/f46d9969fe228dfa8703224fe36c2fcc8cd6540d))
* added Models.list() function ([6c2eae4](https://github.com/googleapis/go-genai/commit/6c2eae47aa6fb60cd2f6ae52744033359e0093ba))
* enable minItem, maxItem, nullable for Schema type when calling Gemini API. ([fb6c8a5](https://github.com/googleapis/go-genai/commit/fb6c8a528b195f07dae7b6130eee059a40d35803))
* enable quick accessor of executable code and code execution result in GenerateContentResponse ([21ca251](https://github.com/googleapis/go-genai/commit/21ca2516b27cbf51b4ab3486da9ca31f3a908204))
* remove ClientConfig.Timeout and add HTTPOptions to ...Config structs ([ba6c431](https://github.com/googleapis/go-genai/commit/ba6c43132ce8a2fcad1fdad48bc3f80b6ecb0a96))
* Support aspect ratio for edit_image ([06d554f](https://github.com/googleapis/go-genai/commit/06d554f78ce4b61cc113f5254c4f5b48415ce25e))
* support edit image and add sample for imagen ([f332cf2](https://github.com/googleapis/go-genai/commit/f332cf26e0c570cd2af4e797a01930ea55b096eb))
* Support Models.EmbedContent function ([a71f0a7](https://github.com/googleapis/go-genai/commit/a71f0a7a181181316e02f4fe21ad6acddae68c1b))


### Bug Fixes

* change int64, float64 types to int32, unit32, float32 to prevent data loss ([af83fa7](https://github.com/googleapis/go-genai/commit/af83fa7501b3e81102b35c1bffd76cdf68203d1b))
* log warning instead of throwing error for GenerateContentResponse.text() quick accessor when there are mixed types of parts. ([006e3af](https://github.com/googleapis/go-genai/commit/006e3af99fb568d89926bb6129b8d890e8f6a0db))


### Miscellaneous Chores

* release 0.5.0 ([14bdd8f](https://github.com/googleapis/go-genai/commit/14bdd8f9b7148c2aa588249415c29396c3b6217c))

## [0.4.0](https://github.com/googleapis/go-genai/compare/v0.3.0...v0.4.0) (2025-02-24)


### Features

* Add Imagen upscale_image support for Go ([8e2afe9](https://github.com/googleapis/go-genai/commit/8e2afe992bae5b30c6d9cd2bfecfc71f12c3f986))
* introduce usability functions to allow quick creation of user content and model content. ([12b5dee](https://github.com/googleapis/go-genai/commit/12b5dee0e6148aa00c5ee3516189e79dc07b1ab8))
* support list all caches in List and All functions ([addc388](https://github.com/googleapis/go-genai/commit/addc3880e38c6026117d91f8019959347469ef12))
* support Models .Get, .Update, .Delete ([e67cd8b](https://github.com/googleapis/go-genai/commit/e67cd8b2d619323bfce97a3b6306521799a6b4f9))


### Bug Fixes

* fix the civil.Date parsing in Citation struct. fixes [#106](https://github.com/googleapis/go-genai/issues/106) ([f530fcf](https://github.com/googleapis/go-genai/commit/f530fcf86fec626bd6bad88c72d26746acada4ff))
* missing context in request. fixes [#104](https://github.com/googleapis/go-genai/issues/104) ([747c5ef](https://github.com/googleapis/go-genai/commit/747c5ef9c781024b0f88f30c77ff382b35f6a52b))
* Remove request body when it's empty. ([cfc82e3](https://github.com/googleapis/go-genai/commit/cfc82e3ca5231506172c9258a1447a114a84ed96))

## [0.3.0](https://github.com/googleapis/go-genai/compare/v0.2.0...v0.3.0) (2025-02-12)


### Features

* Enable Media resolution for Gemini API. ([a22788b](https://github.com/googleapis/go-genai/commit/a22788bb061458bbd15c2fd1a8e2dfdf9e7a3fc8))
* support property_ordering in response_schema (fixes [#236](https://github.com/googleapis/go-genai/issues/236)) ([ac45038](https://github.com/googleapis/go-genai/commit/ac450381046cd673d6a76e04920fc610b182c2c0))

## [0.2.0](https://github.com/googleapis/go-genai/compare/v0.1.0...v0.2.0) (2025-02-05)


### Features

* Add enhanced_prompt to GeneratedImage class ([449f0fb](https://github.com/googleapis/go-genai/commit/449f0fbc1f57b5ce5e20eef587f67f2d0d93a889))
* Add labels for GenerateContent requests ([98231e5](https://github.com/googleapis/go-genai/commit/98231e5e7fa2483004841b50ceee841078e6d951))


### Bug Fixes

* remove unsupported parameter from Gemini API ([39c8868](https://github.com/googleapis/go-genai/commit/39c88682acbf554bad4d7a8ca92a854a7005052a))
* Use camel case for Go function parameters ([94765e6](https://github.com/googleapis/go-genai/commit/94765e68aef1258054711cc601e070e4ef7c80e5))

## [0.1.0](https://github.com/googleapis/go-genai/compare/v0.0.1...v0.1.0) (2025-01-29)


### ⚠ BREAKING CHANGES

* Make some numeric fields to pointer type and bool fields to value type, and rename ControlReferenceTypeControlType* constants

### Features

* [genai-modules][models] Add HttpOptions to all method configs for models. ([765c9b7](https://github.com/googleapis/go-genai/commit/765c9b7311884554c352ec00a0253c2cbbbf665c))
* Add Imagen generate_image support for Go SDK ([068fe54](https://github.com/googleapis/go-genai/commit/068fe541801ced806714662af023a481271402c4))
* Add support for audio_timestamp to types.GenerateContentConfig (fixes [#132](https://github.com/googleapis/go-genai/issues/132)) ([cfede62](https://github.com/googleapis/go-genai/commit/cfede6255a13b4977450f65df80b576342f44b5a))
* Add support for enhance_prompt to model.generate_image ([a35f52a](https://github.com/googleapis/go-genai/commit/a35f52a318a874935a1e615dbaa24bb91625c5de))
* Add ThinkingConfig to generate content config. ([ad73778](https://github.com/googleapis/go-genai/commit/ad73778cf6f1c6d9b240cf73fce52b87ae70378f))
* enable Text() and FunctionCalls() quick accessor for GenerateContentResponse ([3f3a450](https://github.com/googleapis/go-genai/commit/3f3a450954283fa689c9c19a29b0487c177f7aeb))
* Images - Added Image.mime_type ([3333511](https://github.com/googleapis/go-genai/commit/3333511a656b796065cafff72168c112c74de293))
* introducing HTTPOptions to Client ([e3d1d8e](https://github.com/googleapis/go-genai/commit/e3d1d8e6aa0cbbb3f2950c571f5c0a70b7ce8656))
* make Part, FunctionDeclaration, Image, and GenerateContentResponse classmethods argument keyword only ([f7d1043](https://github.com/googleapis/go-genai/commit/f7d1043bb791930d82865a11b83fea785e313922))
* Make some numeric fields to pointer type and bool fields to value type, and rename ControlReferenceTypeControlType* constants ([ee4e5a4](https://github.com/googleapis/go-genai/commit/ee4e5a414640226e9b685a7d67673992f2c63dee))
* support caches create/update/get/update in Go SDK ([0620d97](https://github.com/googleapis/go-genai/commit/0620d97e32b3e535edab8f3f470e08746ace4d60))
* support usability constructor functions for Part struct ([831b879](https://github.com/googleapis/go-genai/commit/831b879ea15a82506299152e9f790f34bbe511f9))


### Miscellaneous Chores

* Released as 0.1.0 ([e046125](https://github.com/googleapis/go-genai/commit/e046125c8b378b5acb05e64ed46c4aac51dd9456))


### Code Refactoring

* rename GenerateImage() to GenerateImage(), rename GenerateImageConfig to GenerateImagesConfig, rename GenerateImageResponse to GenerateImagesResponse, rename GenerateImageParameters to GenerateImagesParameters ([ebb231f](https://github.com/googleapis/go-genai/commit/ebb231f0c86bb30f013301e26c562ccee8380ee0))

## 0.0.1 (2025-01-10)


### Features

* enable response_logprobs and logprobs for Google AI ([#17](https://github.com/googleapis/go-genai/issues/17)) ([51f2744](https://github.com/googleapis/go-genai/commit/51f274411ea770fa8fc16ce316085310875e5d68))
* Go SDK Live module implementation for GoogleAI backend ([f88e65a](https://github.com/googleapis/go-genai/commit/f88e65a7f8fda789b0de5ecc4e2ed9d2bd02cc89))
* Go SDK Live module initial implementation for VertexAI. ([4d82dc0](https://github.com/googleapis/go-genai/commit/4d82dc0c478151221d31c0e3ccde9ac215f2caf2))


### Bug Fixes

* change string type to numeric types ([bfdc94f](https://github.com/googleapis/go-genai/commit/bfdc94fd1b38fb61976f0386eb73e486cc3bc0f8))
* fix README typo ([5ae8aa6](https://github.com/googleapis/go-genai/commit/5ae8aa6deec520f33d1746be411ed55b2b10d74f))
