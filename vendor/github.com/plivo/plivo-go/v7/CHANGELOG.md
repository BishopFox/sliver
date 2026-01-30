# Change Log
## [7.59.2](https://github.com/plivo/plivo-go/tree/v7.59.2) (2025-10-17)
**Feature - Compliance Application rejection_reason field.**

## [7.59.1](https://github.com/plivo/plivo-go/tree/v7.59.1) (2025-09-25)
**Feature - Campaign Error Desciption Field.**

## [7.59.0](https://github.com/plivo/plivo-go/tree/v7.59.0) (2025-04-30)
**Feature - New Param added for Start Recording API.**
- Support `record_channel_type` in Start Recording API and `recordChannelType` in Record XML.

## [7.58.0](https://github.com/plivo/plivo-go/tree/v7.58.0) (2025-04-16)
**Feature - Support record_track_type in MPC Participant Recording Feature.**
- Support record_track_type in MPC Participant Recording Feature.

## [7.57.2](https://github.com/plivo/plivo-go/tree/v7.57.2) (2025-02-24)
**Enhancement - Supporting parameter_name in WhatsApp Template .**
- Supporting parameter_name in WhatsApp Template .

## [7.57.1](https://github.com/plivo/plivo-go/tree/v7.57.1) (2025-01-27)
**New Param - Start Stream Api**
- Support for `cx_bot` parameter in Start Stream API.

## [7.57.0](https://github.com/plivo/plivo-go/tree/v7.57.0) (2024-12-09)
**Feature - MachineDetection params added in Dial XML Element**
- Support for the `machineDetection` parameter in Dial XML Element.

## [7.56.0](https://github.com/plivo/plivo-go/tree/v7.56.0) (2024-11-19)
**Feature - Transcription params  added in  MPC XML Element**
- Support for the `transcriptionUrl`, `transcript`  parameter in MPC XML Element.

## [7.55.1](https://github.com/plivo/plivo-go/tree/v7.55.1) (2024-11-15)
**Feature - RecordParticipantTrack param added in MPC XML creation**
- Support for the `RecordParticipantTrack` parameter in MPC XML Creation.

## [7.55.0](https://github.com/plivo/plivo-go/tree/v7.55.0) (2024-11-12)
**Feature - CreateRecordingTranscription and DeleteRecordingTranscription feature added**
- This API would help in creating transcription for recorded calls for which transcription is not available and delete API to delete.
- Support for the `transcription_url`, `transcript` and `record_participant_track` parameter in MPC Add Participant.

## [7.54.0](https://github.com/plivo/plivo-go/tree/v7.54.0) (2024-10-30)
**Feature - GetRecordingTranscription feature to get transcription**
- Support for the `type` filter parameter, supported filters are transcription, raw and diarized
- Support added for Transcription in MPC

## [7.53.1](https://github.com/plivo/plivo-go/tree/v7.53.1) (2024-10-23)
**Feature - FraudCheck param in Create, Get and List Session**
- Support for the `fraud_check` parameter in sms verify session request
- Added support for `fraud_check` in GET and LIST verify session.

## [7.53.0](https://github.com/plivo/plivo-go/tree/v7.53.0) (2024-10-10)
**Feature - Dtmf param in Create, Get and List Session**
- Support for the `dtmf` parameter in voice verify session request
- Added support for `dtmf` in GET and LIST verify session.
  
## [7.52.0](https://github.com/plivo/plivo-go/tree/v7.52.0) (2024-09-30)
**Feature - Adding new param support for Number Masking session with single party **
- Added `create_session_with_single_party`, `virtual_number_cooloff_period` and `force_pin_authentication` attributes in Masking Session

## [7.51.3](https://github.com/plivo/plivo-go/tree/v7.53.0) (2024-09-06)
**Feature - Adding support for Locale param in Create, Get and List Session**
- Enhance message object
- Added new object param on get and list mdr response: `message_sent_time`, `message_updated_time`, `error_message`

## [7.51.2](https://github.com/plivo/plivo-go/tree/v7.51.2) (2024-09-06)
**Feature - Adding support for brand_name, code_length and app_hash in Create,Get and List Session**
- Added new request param `brand_name`, `code_length` and `app_hash` in create Session API
- Added support for `brand_name` , `app_hash`, `code_length` param in get and list Session response

## [7.51.1](https://github.com/plivo/plivo-go/tree/v7.51.1) (2024-09-05)
**Feature - Adding new element for Audio Stream XML **
- Added `keepCallAlive` element in Audio Stream XML

## [7.51.0](https://github.com/plivo/plivo-go/tree/v7.51.0) (2024-07-11)
**Feature - Adding support for Locale param in Create, Get and List Session**
- Added new request param `locale` in create Session API
- Added support for `locale` param in get and list Session response

## [7.50.1](https://github.com/plivo/plivo-go/tree/v7.50.1) (2024-07-01)
- Added Send digits and send on preanswer attribute in add participant

## [7.50.0](https://github.com/plivo/plivo-go/tree/v7.50.0) (2024-06-20)
- Improving outbound templated message capabilities

## [7.49.2](https://github.com/plivo/plivo-go/tree/v7.49.2) (2024-06-07)
**Bug Fix - List and Get Media object url fix**
- Fixed the media_url response value

## [7.49.1](https://github.com/plivo/plivo-go/tree/v7.49.1) (2024-05-28)
**Feature - Filter support for AppName in ListApplication API**
- Added new filter param 'app_name' in list application api

## [7.49.0](https://github.com/plivo/plivo-go/tree/v7.49.0) (2024-05-20)
**Feature - Adding support for location whatsapp messages**
- Added new param 'location' to [send message API](https://www.plivo.com/docs/sms/api/message#send-a-message) to support location 'whatsapp' messages
- Added new param 'location' in templates to support location based templated messages

## [7.48.0](https://github.com/plivo/plivo-go/tree/v7.48.0) (2024-05-07)
**Feature - Adding support for interactive whatsapp messages**
- Added new param 'interactive' to [send message API](https://www.plivo.com/docs/sms/api/message#send-a-message) to support interactive 'whatsapp' messages

## [7.47.0](https://github.com/plivo/plivo-go/tree/v7.47.0) (2024-05-02)
**Feature - Added SubAccount and GeoMatch for Create Masking Session API of Number Masking.**
-  Added sub_account and geo_match support in MaskingSession APIs

## [7.46.0](https://github.com/plivo/plivo-go/tree/v7.46.0) (2024-04-18)
**Feature - Support for dynamic button components when sending a templated WhatsApp message**
- Added new param `payload` in templates to support dynamic payload in templates

## [7.45.6](https://github.com/plivo/plivo-go/tree/v7.45.6) (2024-04-04)
**Feature - New params for GET and LIST Brand API**
-  Added new param 'declined_reasons' for GET and LIST Brand API

## [7.45.5](https://github.com/plivo/plivo-go/tree/v7.45.5) (2024-02-29)
**Feature - Log Redaction Enhancement**
-  Added log attribute in GET and List MDR response
-  Change log field from bool to string in send SMS 

## [7.45.4](https://github.com/plivo/plivo-go/tree/v7.45.4) (2024-02-26)
**Feature - Added new param 'waitTime' for MPC XML**
-  Added new param 'waitTime' for MPC XML

## [7.45.3](https://github.com/plivo/plivo-go/tree/v7.45.3) (2024-02-12)
**Feature - Added few new PIN related params for Create Masking Session API of Number Masking.**
-  Added few new PIN related params for Create Masking Session API of Number Masking

## [7.45.2](https://github.com/plivo/plivo-go/tree/v7.45.2) (2024-01-25)
**Feature - Added new params 'create_mpc_with_single_participant' for Add Participant API of MPC**
-  Added new params 'create_mpc_with_single_participant' for Add Participant API of MPC

## [7.45.1](https://github.com/plivo/plivo-go/tree/v7.45.1) (2023-12-19)
**Feature - Added type params for speak api in call and mpc**
-  Added params 'type' for POST Speak API for Call and MPC

## [7.45.0](https://github.com/plivo/plivo-go/tree/v7.45.0) (2023-12-14)
**Feature - Added params for GET and LIST Campaign APIs**
-  Added params 'vertical', 'campaign_alias' for GET and LIST Campaign APIs

## [7.44.0](https://github.com/plivo/plivo-go/tree/v7.44.0) (2023-11-20)
**Feature - New params for GET and LIST Campaign API**
-  Added new params 'error_code', 'error_reason' for GET and LIST Campaign API

## [7.43.0](https://github.com/plivo/plivo-go/tree/v7.43.0) (2023-11-29)
**Bug Fix - Create Powerpack and Update Powerpack**
- Create Powerpack and Update Powerpack now take boolean inputs for sticky sender and local connect.

## [7.42.0](https://github.com/plivo/plivo-go/tree/v7.42.0) (2023-11-07)
**Feature - Campaign List API Enhancements**
- registration_status field in LIST API

## [7.41.0](https://github.com/plivo/plivo-go/tree/v7.41.0) (2023-10-31)
**Feature - TollFree Verification API Support**
- API support for Create, Update, Get, Delete and List Tollfree Verification.
- Added New Param `toll_free_sms_verification_id` and `toll_free_sms_verification_order_status `in to the response of the [list all numbers API], [list single number API]
- Added `toll_free_sms_verification_order_status` filter to AccountPhoneNumber - list all my numbers API.

## [7.40.0](https://github.com/plivo/plivo-go/tree/v7.40.0) (2023-10-18)
**Feature - Fixes for Campaign services list API meta data**
- Fixed Meta data response for campaign, brand and profile list

## [7.39.0](https://github.com/plivo/plivo-go/tree/v7.39.0) (2023-10-18)
**Feature - Verify CallerID**
- Added Initiate and Verify VerifyCallerID API
- Added Update, Delete, Get and List verified CallerIDs API

## [7.38.0](https://github.com/plivo/plivo-go/tree/v7.38.0) (2023-10-16)
**Feature - Campaign API Enhancements & New API for Importing Partner Campaigns**
- Import Partner Campaign API
- campaign_source field in LIST / GET API

## [7.37.0](https://github.com/plivo/plivo-go/tree/v7.37.0) (2023-08-25)
**Feature - Added New Param 'carrier_fees', 'carrier_fees_rate', 'destination_network' in Get Message and List Message APIs**
- Added new params on message get and list response

## [7.36.0] (https://github.com/plivo/plivo-go/tree/v7.36.0) (2023-08-10)
**Feature - Verify**
- Added Create Session API
- Added Get Session API
- Added List Session API
- Added Validate Session API

## [7.35.0](https://github.com/plivo/plivo-go/tree/v7.35.0) (2023-08-07)
**Feature - WhatsApp message support**
- Added new param `template` and  new message_type `whatsapp` to [send message API](https://www.plivo.com/docs/sms/api/message#send-a-message)
- Added  new  `message_states` (`read`)   `message_type`(`whatsapp`),`conversation_id`, `conversation_origin`, `conversation_expiry_timestamp` in [list all messages API](https://www.plivo.com/docs/sms/api/message#list-all-messages) and [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message) response


## [7.34.0](https://github.com/plivo/plivo-go/tree/v7.34.0) (2023-08-03)
**Feature - DLT parameters**
- Added new params `DLTEntityID`, `DLTTemplateID`, `DLTTemplateCategory` to the [send message API](https://www.plivo.com/docs/sms/api/message/send-a-message/)
- Added new params `DLTEntityID`, `DLTTemplateID`, `DLTTemplateCategory` to the response for the [list all messages API](https://www.plivo.com/docs/sms/api/message/list-all-messages/) and the [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message)

## [7.33.0](https://github.com/plivo/plivo-go/tree/v7.33.0) (2023-07-31)
**Feature - Number Masking**
- Added Get, Update and List Masking Session API and modified the Create and Delete API
## [7.32.0](https://github.com/plivo/plivo-go/tree/v7.32.0) (2023-06-28)
**Audio Streaming**
- API support for starting, deleting, getting streams on a live call
- XML creation support for stream element


## [7.31.0](https://github.com/plivo/plivo-go/tree/v7.31.0) (2023-06-02)
**Feature - CNAM Lookup**
- Added New Param `cnam_lookup` in to the response of the [list all numbers API], [list single number API]
- Added `cnam_lookup` filter to AccountPhoneNumber - list all my numbers API.
- Added `cnam_lookup` parameter to buy number[Buy a Phone Number]  to configure CNAM Lookup while buying a US number
- Added `cnam_lookup` parameter to update number[Update an account phone number] to configure CNAM Lookup while buying a US number

## [7.30.0](https://github.com/plivo/plivo-go/tree/v7.24.1) (2023-03-16)
**Feature - Added a new param in getCallDetails api**
- From now on we can see CNAM (caller_id name) details at CDR level.

## [7.29.0](https://github.com/plivo/plivo-java/tree/v7.29.0) (2023-05-29)
**Feature - Recording API changes**
- Added `monthly_recording_storage_amount`, `recording_storage_rate`, `rounded_recording_duration`, and `recording_storage_duration` parameters to the response for [get single recording API](https://www.plivo.com/docs/voice/api/recording#retrieve-a-recording) and [get all recordings API](https://www.plivo.com/docs/voice/api/recording#list-all-recordings)
- Added `recording_storage_duration` parameter as a filter option for [get all recordings API](https://www.plivo.com/docs/voice/api/recording#list-all-recordings)

## [7.28.0](https://github.com/plivo/plivo-go/tree/v7.28.0) (2023-05-04)
**Feature - Added New Param 'renewalDate' in Get Number and List Numbers APIs**
- Add New Param `renewalDate` to the response of the [list all numbers API], [list single number API]
- Add 3 new filters to AccountPhoneNumber - list all my numbers API:`renewal_date`, `renewal_date__gt`, `renewal_date__gte`,`renewal_date__lt` and `renewal_date__lte` (https://www.plivo.com/docs/numbers/api/account-phone-number#list-all-my-numbers)

## [7.27.0](https://github.com/plivo/plivo-go/tree/v7.27.0) (2023-04-25)
**Feature - Added New Param 'replaced_sender' in Get Message and List Message APIs**
- Add `replaced_sender` to the response for the [list all messages API](https://www.plivo.com/docs/sms/api/message/list-all-messages/) and the [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message)
- Add `api_id` to the response for the get message details API

## [7.26.0](https://github.com/plivo/plivo-go/tree/v7.26.0) (2023-04-11)
**Feature - Added New Param 'source_ip' in GetCall and ListCalls**
- Added `source_ip` to the response for the [retrieve a call details API](https://www.plivo.com/docs/voice/api/call#retrieve-a-call) and the [retreive all call details API]

## [7.25.0](https://github.com/plivo/plivo-go/tree/v7.25.0) (2023-17-03)
- Added New Param `created_at` to the response for the [list all profiles API](https://www.plivo.com/docs/sms/api/10dlc/profile#retrieve-all-profiles) and the [get profile API](https://www.plivo.com/docs/sms/api/10dlc/profile#retrieve-a-specific-profile) and the [list all brands API](https://www.plivo.com/docs/sms/api/10dlc/brand#retrieve-all-brands) and the [get brand API](https://www.plivo.com/docs/sms/api/10dlc/brand#retrieve-a-specific-brand)
and the [list all campaigns API](https://www.plivo.com/docs/sms/api/10dlc/campaign#retrieve-all-campaigns) and the [get campaign API](https://www.plivo.com/docs/sms/api/10dlc/campaign#retrieve-a-specific-campaign)

## [7.24.0](https://github.com/plivo/plivo-go/tree/v7.24.0) (2023-03-14)
**Fix - Add fix for CVE-2020-26160 and CVE-2022-32149**
- Upgrade dependencies to fix security vulnerabilities.

## [7.23.0](https://github.com/plivo/plivo-go/tree/v7.23.0) (2023-03-03)
**Feature - Added New Param 'is_domestic' in Get Message and List Message APIs**
- Add `is_domestic` to the response for the [list all messages API](https://www.plivo.com/docs/sms/api/message/list-all-messages/) and the [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message)

## [7.22.0](https://github.com/plivo/plivo-go/tree/v7.22.0) (2023-02-23)
**Feature - Enhance MDR filtering capabilities **
- Added new fields on MDR object response

## [7.21.0](https://github.com/plivo/plivo-go/tree/v7.21.0) (2023-02-21)
**Feature - MPC Speak API**
- Added functionality to start and stop Speak in an MPC

## [7.20.0](https://github.com/plivo/plivo-go/tree/v7.20.0) (2023-02-20)
**Feature - MPC API**
- Added support for agent_hold_nusic and customer_hold_music in the XML generation

## [7.19.0](https://github.com/plivo/plivo-go/tree/v7.19.0) (2023-02-16)
**Feature - MPC AddParticipant API**
- Added two new param - agent_hold_music and customer_hold_music in AddParticipant API

## [7.18.0](https://github.com/plivo/plivo-go/tree/v7.18.0) (2023-01-25)
**Feature - Added New Param 'requester_ip' in Get Message and List Mssage APIs**
- Add `requester_ip` to the response for the [list all messages API](https://www.plivo.com/docs/sms/api/message/list-all-messages/) and the [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message)

## [7.17.1](https://github.com/plivo/plivo-go/tree/v7.17.1) (2023-01-18)
**Feature - Adding new param 'message_expiry' in Send Message API**
-  Added new param 'message_expiry' in Send Message API 


## [7.17.0](https://github.com/plivo/plivo-go/tree/v7.17.0) (2023-01-10)
**Feature - Number Masking**
- Added Create and Delete Masking Session API

## [7.16.0](https://github.com/plivo/plivo-go/tree/v7.16.0) (2022-12-16)
**Feature - Update campaign**
- Update campaign API

## [7.15.0](https://github.com/plivo/plivo-go/tree/v7.15.0) (2022-12-06)
**Feature - Delete campaign and brand API**
- Added Delete campaign and brand API

## [7.14.0](https://github.com/plivo/plivo-go/tree/v7.14.0) (2022-10-17)
**Feature - Brandusecase API, 10DLC api enhancements**
- Added Brandusecase API, 10DLC api enhancements

## [7.13.0](https://github.com/plivo/plivo-go/tree/v7.13.0) (2022-10-14)
**Adding new attributes to Account PhoneNumber object**
-Added 3 new keys to AccountPhoneNumber object:`tendlc_registration_status`, `tendlc_campaign_id` and `toll_free_sms_verification` (https://www.plivo.com/docs/numbers/api/account-phone-number#the-accountphonenumber-object)
-Added 3 new filters to AccountPhoneNumber - list all my numbers API:`tendlc_registration_status`, `tendlc_campaign_id` and `toll_free_sms_verification` (https://www.plivo.com/docs/numbers/api/account-phone-number#list-all-my-numbers)

## [7.12.1](https://github.com/plivo/plivo-go/tree/v7.12.1) (2022-09-28)
**Adding more attributes to campaign creation**
- Adding more attributes to campaign creation request

## [7.12.0](https://github.com/plivo/plivo-go/tree/v7.12.0) (2022-08-30)
**Feature - 10DLC api updates**
- Updated 10dlc api with total 15 apis now such as campaign, brand, profile and number link

## [7.11.0](https://github.com/plivo/plivo-go/tree/v7.11.0) (2022-08-01)
**Feature - Token creation**
- `JWT Token Creation API` added functionality to create a new JWT token.

## [7.10.0](https://github.com/plivo/plivo-go/tree/v7.10.0) (2022-07-11)
**Feature - STIR Attestation**
- Add stir attestation param as part of Get CDR and Get live call APIs Response

## [7.9.0](https://github.com/plivo/plivo-go/tree/v7.9.0) (2022-05-05)
**Features - List all recordings and The MultiPartyCall element**
- `fromNumber` and `toNumber` added to filtering param [List all recordings](https://www.plivo.com/docs/voice/api/recording#list-all-recordings)
- `recordMinMemberCount` param added in [Add a participant to a multiparty call using API](https://www.plivo.com/docs/voice/api/multiparty-call/participants#add-a-participant)

## [7.8.0](https://github.com/plivo/plivo-go/tree/v7.8.0) (2022-03-25)
**Features - DialElement**
- `confirmTimeout` parameter added to [The Dial element](https://www.plivo.com/docs/voice/xml/dial/)

## [7.7.2](https://github.com/plivo/plivo-go/tree/v7.7.2) (2022-03-23)
**Bug Fix - Voice**
- Added `Polly.Marlene` to [SSML voices](https://www.plivo.com/docs/voice/concepts/ssml#ssml-voices)

## [7.7.1](https://github.com/plivo/plivo-go/tree/v7.7.1) (2022-03-17)
**Bug Fix - Voice**
- Added `machine_detection_url` and `machine_detection_method` in [Make a call API](https://www.plivo.com/docs/voice/api/call#make-a-call)

## [7.7.0](https://github.com/plivo/plivo-go/tree/v7.7.0) (2022-03-02)
**Bug Fix - Fix go modules**
- Fix the import path for go modules to work

## [7.6.1](https://github.com/plivo/plivo-go/tree/v7.6.1) (2022-02-22)
**Features - ListParticipants**
- Parameter added as member_address in response and mock

## [7.6.0](https://github.com/plivo/plivo-go/tree/v7.6.0) (2022-01-27)
**Features - MPCStartRecording**
- Parameter name change from statusCallBack to recordingCallback

## [7.5.0](https://github.com/plivo/plivo-go/tree/v7.5.0) (2021-12-14)
**Features - Voice**
- Routing SDK traffic through Akamai endpoints for all the [Voice APIs](https://www.plivo.com/docs/voice/api/overview/)

## [7.4.0](https://github.com/plivo/plivo-go/tree/v7.4.0) (2021-12-02)
**Features - Messaging: 10 DLC**
- 10DLC API's for brand and campaign support.

## [7.3.0](https://github.com/plivo/plivo-go/tree/v7.3.0) (2021-11-23)
**Features - Voice: Multiparty calls**
- The [Add Multiparty Call API](https://www.plivo.com/docs/voice/api/multiparty-call/participants#add-a-participant) allows for greater functionality by accepting options like `start recording audio`, `stop recording audio`, and their HTTP methods.
- [Multiparty Calls](https://www.plivo.com/docs/voice/api/multiparty-call/) now has new APIs to `stop` and `play` audio.


## [7.2.2](https://github.com/plivo/plivo-go/tree/v7.2.2) (2021-07-29)
- Removed validation for `ringtimeout` and `delaydial` params in [Start a multi party call](https://www.plivo.com/docs/voice/api/multiparty-call#start-a-new-multiparty-call).

## [7.2.1](https://github.com/plivo/plivo-go/tree/v7.2.1) (2021-07-22)
- Updated default HTTP client request timeout to 5 seconds.

## [7.2.0](https://github.com/plivo/plivo-go/tree/v7.2.0) (2021-07-14)
- Add SDK support for Voice MultiPartyCall APIs and XML.

## [7.1.0](https://github.com/plivo/plivo-go/tree/v7.1.0) (2021-07-13)
- Power pack ID has been included to the response for the [list all messages API](https://www.plivo.com/docs/sms/api/message/list-all-messages/) and the [get message details API](https://www.plivo.com/docs/sms/api/message#retrieve-a-message).
- Support for filtering messages by Power pack ID has been added to the [list all messages API](https://www.plivo.com/docs/sms/api/message#list-all-messages).


## [7.0.0](https://github.com/plivo/plivo-go/tree/v7.0.0) (2021-07-05)
- **BREAKING**: Remove the total_count parameter in meta data for list MDR response

## [6.0.1](https://github.com/plivo/plivo-go/tree/v6.0.1) (2021-07-02)
- Read voice network group from voice pricing
- Fix GetCDR and ListCDR response to include all fields

## [6.0.0](https://github.com/plivo/plivo-go/tree/v6.0.0) (2021-06-29)
- **BREAKING**: Update AddSpeak method signature: remove optional parameters
- Add methods to set SpeakElement attributes

## [5.6.0](https://github.com/plivo/plivo-go/tree/v5.6.0) (2021-06-15)
- Add stir verification param as part of Get CDR and live call APIs

## [5.5.2](https://github.com/plivo/plivo-go/tree/v5.5.2) (2021-04-08)
- Read origination prefix from voice pricing

## [5.5.1](https://github.com/plivo/plivo-go/tree/v5.5.1) (2020-12-16)
- Add SSML utilities

## [5.5.0](https://github.com/plivo/plivo-go/tree/v5.5.0) (2020-11-17)
- Add number_priority support for Powerpack API.

## [5.4.1](https://github.com/plivo/plivo-go/tree/v5.4.1) (2020-11-11)
- Fix send SMS json payload.

## [5.4.0](https://github.com/plivo/plivo-go/tree/v5.4.0) (2020-11-05)
- Add Regulatory Compliance API support.

## [5.3.0](https://github.com/plivo/plivo-go/tree/v5.3.0) (2020-10-31)
- Change lookup API endpoint and response.

## [5.2.0](https://github.com/plivo/plivo-go/tree/v5.2.0) (2020-09-25)
- Add Lookup API support.

## [5.1.0](https://github.com/plivo/plivo-go/tree/v5.1.0) (2020-09-24)
- Add "publicURI" optional param support for Application API.

## [5.0.0](https://github.com/plivo/plivo-go/tree/v5.0.0) (2020-08-19)
- Internal changes in Phlo for MultiPartyCall component.
- **BREAKING**: Rename MultiPartyCall struct to PhloMultiPartyCall.

## [4.9.1](https://github.com/plivo/plivo-go/tree/v4.9.1) (2020-08-10)
- Fix Get Details of a Call API response.

## [4.9.0](https://github.com/plivo/plivo-go/tree/v4.9.0) (2020-08-04)
- Add service type support(SMS/MMS) for Powerpack API.

## [4.8.0](https://github.com/plivo/plivo-go/tree/v4.8.0) (2020-07-23)
- Add retries to multiple regions for voice requests.

## [4.7.1](https://github.com/plivo/plivo-go/tree/v4.7.1) (2020-07-13)
- Fix Call Create & Retrieve Call details API responses.

## [4.7.0](https://github.com/plivo/plivo-go/tree/v4.7.0) (2020-05-28)
- Add JWT helper functions.

## [4.6.0](https://github.com/plivo/plivo-go/tree/v4.6.0) (2020-04-29)
- Add V3 signature helper functions.

## [4.5.0](https://github.com/plivo/plivo-go/tree/v4.5.0) (2020-04-24)
- Add city and mms filter support for Number Search API
- Add city, country and mms into List Number and Number Search API's Response
- Fix for TotalCount in Number API's Response

## [4.4.0](https://github.com/plivo/plivo-go/tree/v4.4.0) (2020-03-31)
- Add application cascade delete support.

## [4.3.0](https://github.com/plivo/plivo-go/tree/v4.3.0) (2020-03-30)
- Add Tollfree support for Powerpack

## [4.2.0](https://github.com/plivo/plivo-go/tree/v4.2.0) (2020-03-27)
- Add post call quality feedback API support.

## [4.1.6](https://github.com/plivo/plivo-go/tree/v4.1.6) (2020-02-25)
- Add Media support

## [4.1.5](https://github.com/plivo/plivo-go/tree/v4.1.5) (2020-01-24)
- Hot fix for go build

## [4.1.4](https://github.com/plivo/plivo-go/tree/v4.1.4) (2019-12-20)
- Add Powerpack support

## [4.1.3](https://github.com/plivo/plivo-go/tree/v4.1.3) (2019-12-04)
- Add MMS support

## [4.1.2](https://github.com/plivo/plivo-go/tree/v4.1.2) (2019-11-13)
- Add GetInput XML support

## [4.1.1](https://github.com/plivo/plivo-go/tree/v4.1.1) (2019-11-05)
- Add SSML support

## [4.1.0](https://github.com/plivo/plivo-go/tree/v4.1.0) (2019-03-12)
- Add PHLO support
- Add Multi-party call triggers

## [4.0.6-beta1](https://github.com/plivo/plivo-go/tree/v4.0.6-beta1) (2019-02-05)
- Add PHLO support in beta
- Add Multi-party call triggers

## [4.0.5](https://github.com/plivo/plivo-go/tree/v4.0.5) (2018-11-19)
- Add hangup party details to CDR. CDR filtering allowed by hangup_source and hangup_cause_code.
- Add sub-account cascade delete support.
- Add call status to GET calls and live-calls methods.

## [4.0.4](https://github.com/plivo/plivo-go/tree/v4.0.4) (2018-10-31)
- Add live calls filtering by to, from numbers and call direction.

## [4.0.3](https://github.com/plivo/plivo-go/tree/v4.0.3) (2018-10-01)
- Added Trackable parameter in messages.

## [4.0.2](https://github.com/plivo/plivo-go/tree/v4.0.2) (2018-09-18)
- Added parent_call_uuid parameter to filter calls.
- Queued status added for filtering calls in queued status.
- Added log_incoming_messages parameter to application create and update.
- Added powerpack support.

## [4.0.1](https://github.com/plivo/plivo-go/tree/v4.0.1) (2018-07-25)
- Fixed caller id retrieval

## [4.0.0](https://github.com/plivo/plivo-go/tree/v4.0.0) (2018-01-18)
- Major restructuring of the repo to bring all go files to repo's root
- Supports v2 signature validation
- A few fixes (#2 & #3)

## [v4.0.0-beta.1](https://github.com/plivo/plivo-go/releases/tag/v4.0.0-beta.1) (2017-10-25)
- The official SDK of Plivo
- Supports all Go versions >= 1.0.x
