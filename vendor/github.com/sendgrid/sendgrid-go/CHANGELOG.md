# Change Log
All notable changes to this project will be documented in this file.

[2025-05-29] Version 3.16.1
---------------------------
**Library - Chore**
- [PR #496](https://github.com/sendgrid/sendgrid-go/pull/496): update licence year. Thanks to [@tiwarishubham635](https://github.com/tiwarishubham635)!


[2024-08-26] Version 3.16.0
---------------------------
**Library - Chore**
- [PR #479](https://github.com/sendgrid/sendgrid-go/pull/479): updates for manual release. Thanks to [@sbansla](https://github.com/sbansla)!
- [PR #477](https://github.com/sendgrid/sendgrid-go/pull/477): fixed failed test cases due to go upgrade. Thanks to [@sbansla](https://github.com/sbansla)!

**Library - Feature**
- [PR #471](https://github.com/sendgrid/sendgrid-go/pull/471): add mail_v3 functionality for reply_to_list. Thanks to [@lopezator](https://github.com/lopezator)!


[2024-08-08] Version 3.15.0
---------------------------
**Library - Feature**
- [PR #471](https://github.com/sendgrid/sendgrid-go/pull/471): add mail_v3 functionality for reply_to_list


[2023-12-01] Version 3.14.0
---------------------------
**Library - Chore**
- [PR #470](https://github.com/sendgrid/sendgrid-go/pull/470): removed SetHost and shifted SetDataResidency to sendgrid.go. Thanks to [@tiwarishubham635](https://github.com/tiwarishubham635)!

**Library - Feature**
- [PR #469](https://github.com/sendgrid/sendgrid-go/pull/469): added data residency for eu and global regions. Thanks to [@tiwarishubham635](https://github.com/tiwarishubham635)!


[2023-08-10] Version 3.13.0
---------------------------
**Library - Feature**
- [PR #468](https://github.com/sendgrid/sendgrid-go/pull/468): gzip mail body when content-encoding is set to gzip. Thanks to [@Bankq](https://github.com/Bankq)!


[2022-09-21] Version 3.12.0
---------------------------
**Library - Feature**
- [PR #464](https://github.com/sendgrid/sendgrid-go/pull/464): go 1.19 compatibility. Thanks to [@AlaricWhitney](https://github.com/AlaricWhitney)!

**Library - Test**
- [PR #462](https://github.com/sendgrid/sendgrid-go/pull/462): Adding misc as PR type. Thanks to [@rakatyal](https://github.com/rakatyal)!

**Library - Docs**
- [PR #459](https://github.com/sendgrid/sendgrid-go/pull/459): Modify README.md in alignment with SendGrid Support. Thanks to [@garethpaul](https://github.com/garethpaul)!


[2022-03-09] Version 3.11.1
---------------------------
**Library - Chore**
- [PR #456](https://github.com/sendgrid/sendgrid-go/pull/456): push Datadog Release Metric upon deploy success. Thanks to [@eshanholtz](https://github.com/eshanholtz)!


[2022-02-09] Version 3.11.0
---------------------------
**Library - Feature**
- [PR #443](https://github.com/sendgrid/sendgrid-go/pull/443): Refactor Inbound package to provide access to SendGrid's pre-processing. Thanks to [@qhenkart](https://github.com/qhenkart)!

**Library - Docs**
- [PR #454](https://github.com/sendgrid/sendgrid-go/pull/454): add docs for bypass mail options. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!

**Library - Chore**
- [PR #453](https://github.com/sendgrid/sendgrid-go/pull/453): upgrade supported language versions. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #452](https://github.com/sendgrid/sendgrid-go/pull/452): merge test and gh release workflows. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!


[2022-01-12] Version 3.10.5
---------------------------
**Library - Chore**
- [PR #449](https://github.com/sendgrid/sendgrid-go/pull/449): update license year. Thanks to [@JenniferMah](https://github.com/JenniferMah)!


[2021-12-15] Version 3.10.4
---------------------------
**Library - Chore**
- [PR #448](https://github.com/sendgrid/sendgrid-go/pull/448): migrate to Github actions. Thanks to [@beebzz](https://github.com/beebzz)!


[2021-10-18] Version 3.10.3
---------------------------
**Library - Docs**
- [PR #440](https://github.com/sendgrid/sendgrid-go/pull/440): update signed webhook usage documentation. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!


[2021-10-06] Version 3.10.2
---------------------------
**Library - Chore**
- [PR #436](https://github.com/sendgrid/sendgrid-go/pull/436): Remove mail.send helpers with on-behalf-of header. Thanks to [@bjohnson-va](https://github.com/bjohnson-va)!


[2021-09-22] Version 3.10.1
---------------------------
**Library - Chore**
- [PR #438](https://github.com/sendgrid/sendgrid-go/pull/438): add support for 1.16. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!


[2021-05-05] Version 3.10.0
---------------------------
**Library - Feature**
- [PR #433](https://github.com/sendgrid/sendgrid-go/pull/433): support context.Context. Thanks to [@johejo](https://github.com/johejo)!


[2021-04-21] Version 3.9.0
--------------------------
**Library - Feature**
- [PR #430](https://github.com/sendgrid/sendgrid-go/pull/430): add Email Length validation. Thanks to [@itsksaurabh](https://github.com/itsksaurabh)!


[2021-02-10] Version 3.8.0
--------------------------
**Library - Fix**
- [PR #426](https://github.com/sendgrid/sendgrid-go/pull/426): typo in method name. Thanks to [@thinkingserious](https://github.com/thinkingserious)!
- [PR #355](https://github.com/sendgrid/sendgrid-go/pull/355): content value issue by implementing NewSingleEmailPlanText. Thanks to [@prakashpandey](https://github.com/prakashpandey)!
- [PR #398](https://github.com/sendgrid/sendgrid-go/pull/398): Add error handling for upstream on inbound parse. Thanks to [@thavanle](https://github.com/thavanle)!

**Library - Feature**
- [PR #425](https://github.com/sendgrid/sendgrid-go/pull/425): Add support for more bypass settings. Thanks to [@yousifh](https://github.com/yousifh)!


[2020-11-18] Version 3.7.2
--------------------------
**Library - Docs**
- [PR #281](https://github.com/sendgrid/sendgrid-go/pull/281): Email activity API Documentation. Thanks to [@dhoeric](https://github.com/dhoeric)!


[2020-11-05] Version 3.7.1
--------------------------
**Library - Test**
- [PR #411](https://github.com/sendgrid/sendgrid-go/pull/411): ensure source files are properly formatted. Thanks to [@childish-sambino](https://github.com/childish-sambino)!

**Library - Fix**
- [PR #415](https://github.com/sendgrid/sendgrid-go/pull/415): Rename LICENSE.md to LICENSE. Thanks to [@coolaj86](https://github.com/coolaj86)!

**Library - Docs**
- [PR #282](https://github.com/sendgrid/sendgrid-go/pull/282): Update examples using inline attachment with ContentID. Thanks to [@anchepiece](https://github.com/anchepiece)!


[2020-10-14] Version 3.7.0
--------------------------
**Library - Feature**
- [PR #410](https://github.com/sendgrid/sendgrid-go/pull/410): allow personalization of From name and email for each recipient. Thanks to [@JenniferMah](https://github.com/JenniferMah)!

**Library - Fix**
- [PR #272](https://github.com/sendgrid/sendgrid-go/pull/272): Accept empty html on Email helper NewSingleEmail(). Thanks to [@tjun](https://github.com/tjun)!


[2020-09-28] Version 3.6.4
--------------------------
**Library - Fix**
- [PR #408](https://github.com/sendgrid/sendgrid-go/pull/408): don't wrap names in double-quotes. Thanks to [@childish-sambino](https://github.com/childish-sambino)!


[2020-09-02] Version 3.6.3
--------------------------
**Library - Docs**
- [PR #287](https://github.com/sendgrid/sendgrid-go/pull/287): Correct *.md files using Grammarly. Thanks to [@vkartik97](https://github.com/vkartik97)!


[2020-08-19] Version 3.6.2
--------------------------
**Library - Chore**
- [PR #402](https://github.com/sendgrid/sendgrid-go/pull/402): update GitHub branch references to use HEAD. Thanks to [@thinkingserious](https://github.com/thinkingserious)!


[2020-08-05] Version 3.6.1
--------------------------
**Library - Docs**
- [PR #329](https://github.com/sendgrid/sendgrid-go/pull/329): Remove references to legacy "Whitelabel" Verbiage. Thanks to [@crweiner](https://github.com/crweiner)!

**Library - Fix**
- [PR #401](https://github.com/sendgrid/sendgrid-go/pull/401): use the last version of testify that works for older go versions. Thanks to [@childish-sambino](https://github.com/childish-sambino)!

**Library - Chore**
- [PR #400](https://github.com/sendgrid/sendgrid-go/pull/400): migrate to new default sendgrid-oai branch. Thanks to [@eshanholtz](https://github.com/eshanholtz)!


[2020-05-14] Version 3.6.0
--------------------------
**Library - Feature**
- [PR #392](https://github.com/sendgrid/sendgrid-go/pull/392): add support for Twilio Email. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #390](https://github.com/sendgrid/sendgrid-go/pull/390): add function for signature verification. Thanks to [@brpat07](https://github.com/brpat07)!
- [PR #389](https://github.com/sendgrid/sendgrid-go/pull/389): add support and example for secure webhook feature. Thanks to [@brpat07](https://github.com/brpat07)!

**Library - Fix**
- [PR #388](https://github.com/sendgrid/sendgrid-go/pull/388): refactor and fix inbound email handling. Thanks to [@eshanholtz](https://github.com/eshanholtz)!
- [PR #391](https://github.com/sendgrid/sendgrid-go/pull/391): migrate to common prism setup. Thanks to [@childish-sambino](https://github.com/childish-sambino)!


[2020-04-01] Version 3.5.4
--------------------------
**Library - Docs**
- [PR #386](https://github.com/sendgrid/sendgrid-go/pull/386): support verbiage for login issues. Thanks to [@adamchasetaylor](https://github.com/adamchasetaylor)!


[2020-02-19] Version 3.5.3
--------------------------
**Library - Docs**
- [PR #295](https://github.com/sendgrid/sendgrid-go/pull/295): Update documentation for retrieving a list of all templates. Thanks to [@renshuki](https://github.com/renshuki)!


[2020-02-05] Version 3.5.2
--------------------------
**Library - Docs**
- [PR #309](https://github.com/sendgrid/sendgrid-go/pull/309): Fixed link to bug report template. Thanks to [@alxshelepenok](https://github.com/alxshelepenok)!

**Library - Chore**
- [PR #372](https://github.com/sendgrid/sendgrid-go/pull/372): Add current Go versions to Travis. Thanks to [@pangaunn](https://github.com/pangaunn)!


[2020-01-30] Version 3.5.1
--------------------------
**Library - Chore**
- [PR #382](https://github.com/sendgrid/sendgrid-go/pull/382): clean up prism installation. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #379](https://github.com/sendgrid/sendgrid-go/pull/379): prep repo for automation. Thanks to [@thinkingserious](https://github.com/thinkingserious)!

**Library - Docs**
- [PR #380](https://github.com/sendgrid/sendgrid-go/pull/380): baseline all the templated markdown docs. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #348](https://github.com/sendgrid/sendgrid-go/pull/348): fix usage link in README. Thanks to [@BogdanHabic](https://github.com/BogdanHabic)!

**Library - Fix**
- [PR #353](https://github.com/sendgrid/sendgrid-go/pull/353): double quote escape names with special characters. Thanks to [@haleyrc](https://github.com/haleyrc)!


[2019-06-13] Version 3.5.0
--------------------------
### Added
- [PR #117](https://github.com/sendgrid/sendgrid-go/pull/117): Add release notifications. Big thanks to [Gabriel Krell](https://github.com/gabrielkrell) for the PR!
- [PR #118](https://github.com/sendgrid/sendgrid-go/pull/118): Update USE_CASES.md formatting. Big thanks to [Kyle Roberts](https://github.com/kylearoberts) for the PR!
- [PR #123](https://github.com/sendgrid/sendgrid-go/pull/123): Update USE_CASES.md with substitutions and sections. Big thanks to [Kyle Roberts](https://github.com/kylearoberts) for the PR!
- [PR #111](https://github.com/sendgrid/sendgrid-go/pull/111): Add examples from "Personalizations Example Index" to USE_CASES.md. Big thanks to [Christopher Li](https://github.com/LiYChristopher) for the PR!
- [PR #127](https://github.com/sendgrid/sendgrid-go/pull/127): Update Travis YML to use newer go versions. Big thanks to [Tariq Ibrahim](https://github.com/tariq1890) for the PR!
- [PR #143](https://github.com/sendgrid/sendgrid-go/pull/143): Added a warning about the error return from sendgrid.API in TROUBLESHOOTING.md. Big thanks to [Leandro Lugaresi](https://github.com/leandro-lugaresi) for the PR!
- [PR #128](https://github.com/sendgrid/sendgrid-go/pull/128): Added a Mail Refactor proposal. Big thanks to [Suchit Parikh](https://github.com/suchitparikh) for the PR!
- [PR #153](https://github.com/sendgrid/sendgrid-go/pull/153): Added Code of Conduct. Big thanks to [Rubemlrm](https://github.com/Rubemlrm) for the PR!
- [PR #139](https://github.com/sendgrid/sendgrid-go/pull/139): Added attachment use case examples. Big thanks to [Christopher Li](https://github.com/LiYChristopher) for the PR!
- [PR #165](https://github.com/sendgrid/sendgrid-go/pull/165): Update USE_CASES.md with statistics and domain whitelabel examples. Big thanks to [Nexus Web Development](https://github.com/NexWeb) for the PR!
- [PR #186](https://github.com/sendgrid/sendgrid-go/pull/186): Moved logo to top and added more badges. Big thanks to [Alex](https://github.com/myzeprog) for the PR!
- [PR #187](https://github.com/sendgrid/sendgrid-go/pull/187): Made the README/Doc sections more SEO friendly. Big thanks to [Alex](https://github.com/myzeprog) for the PR!
- [PR #188](https://github.com/sendgrid/sendgrid-go/pull/188): Add Go specific badges to the README. Big thanks to [Thorsten Schifferdecker](https://github.com/curx) for the PR!
- [PR #181](https://github.com/sendgrid/sendgrid-go/pull/181): Add review request body section to TROUBLESHOOTING.md. Big thanks to [Alex](https://github.com/myzeprog) for the PR!
- [PR #363](https://github.com/sendgrid/sendgrid-go/pull/363): Twilio branding + CLA updates.
- [PR #217](https://github.com/sendgrid/sendgrid-go/pull/217): Initialize map on declaration (round 2). Big thanks to [Noah Santschi-Cooney](https://github.com/Strum355) for the PR!
- [PR #216](https://github.com/sendgrid/sendgrid-go/pull/216): Initialize map on declaration. Big thanks to [Noah Santschi-Cooney](https://github.com/Strum355) for the PR!
- [PR #210](https://github.com/sendgrid/sendgrid-go/pull/210): Add github PR template. Big thanks to [Alex](https://github.com/pushkyn) for the PR!
- [PR #225](https://github.com/sendgrid/sendgrid-go/pull/225): Add test for license date range. Big thanks to [Mansour Rahimi](https://github.com/m4ns0ur) for the PR!
- [PR #214](https://github.com/sendgrid/sendgrid-go/pull/214): Add a .env_sample file, update gitignore, update README.md. Big thanks to [thepriefy](https://github.com/thepriefy) for the PR!
- [PR #137](https://github.com/sendgrid/sendgrid-go/pull/137): Add Dockerize. Big thanks to [Eric Ho](https://github.com/dhoeric) for the PR!
- [PR #200](https://github.com/sendgrid/sendgrid-go/pull/200): Helping get golint to 100%. Big thanks to [Paul Lovato](https://github.com/Cleanse) for the PR!
- [PR #234](https://github.com/sendgrid/sendgrid-go/pull/234): Add announcement: Software Engineer role. Big thanks to [Marghodk](https://github.com/Marghodk) for the PR!
- [PR #228](https://github.com/sendgrid/sendgrid-go/pull/228): Include Gometalinter in Travis CI build. Big thanks to [Vasko Zdravevski](https://github.com/vaskoz) for the PR!
- [PR #229](https://github.com/sendgrid/sendgrid-go/pull/229): Add test for checking specific repo files. Big thanks to [Mansour Rahimi](https://github.com/m4ns0ur) for the PR!
- [PR #231](https://github.com/sendgrid/sendgrid-go/pull/231): Adds codecov. Big thanks to [Charlie Lewis](https://github.com/cglewis) for the PR!
- [PR #155](https://github.com/sendgrid/sendgrid-go/pull/155): Added optional rate limit handling. Big thanks to [Andy Trimble](https://github.com/andy-trimble) for the PR!
- [PR #250](https://github.com/sendgrid/sendgrid-go/pull/250): Exclude time.Until lint error until we stop supporting Go 1.7 and lower. Big thanks to [Dustin Mowcomber](https://github.com/dmowcomber) for the PR!
- [PR #264](https://github.com/sendgrid/sendgrid-go/pull/264): Readability update. Big thanks to [Anshul Singhal](https://github.com/af4ro) for the PR!
- [PR #263](https://github.com/sendgrid/sendgrid-go/pull/263): Dynamic template support. Big thanks to [Devin Chasanoff](https://github.com/devchas) for the PR!
- [PR #268](https://github.com/sendgrid/sendgrid-go/pull/268): mail: add test case on empty HTML to NewSingleEmail. Big thanks to [Arthur Silva](https://github.com/arxdsilva) for the PR!
- [PR #269](https://github.com/sendgrid/sendgrid-go/pull/269): use testify. Big thanks to [Arthur Silva](https://github.com/arxdsilva) for the PR!
- [PR #280](https://github.com/sendgrid/sendgrid-go/pull/280): helpers/mail: add testify to new test. Big thanks to [Arthur Silva](https://github.com/arxdsilva) for the PR!
- [PR #194](https://github.com/sendgrid/sendgrid-go/pull/194): Allows users to submit rfc822 formatted email addresses. Big thanks to [Tariq Ibrahim](https://github.com/tariq1890) for the PR!
- [PR #197](https://github.com/sendgrid/sendgrid-go/pull/197): Make Getenv("message") parameter more professional. Big thanks to [Nafis Faysal](https://github.com/nafisfaysal) for the PR!
- [PR #238](https://github.com/sendgrid/sendgrid-go/pull/238): Added Code Review to Contributing.md. Big thanks to [Manjiri Tapaswi](https://github.com/mptap) for the PR!
- [PR #293](https://github.com/sendgrid/sendgrid-go/pull/293): Use case directory structure update. Big thanks to [Arshad Kazmi](https://github.com/arshadkazmi42) for the PR!
- [PR #243](https://github.com/sendgrid/sendgrid-go/pull/243): Add the ability to impersonate a subuser. Big thanks to [Boris M](https://github.com/denwwer) for the PR!
- [PR #327](https://github.com/sendgrid/sendgrid-go/pull/327): Update prerequisites verbiage. Big thanks to [Rishabh](https://github.com/Rishabh04-02) for the PR!

### Fixed
- [PR #141](https://github.com/sendgrid/sendgrid-go/pull/141): Fix TROUBLESHOOTING.md typo. Big thanks to [Cícero Pablo](https://github.com/ciceropablo) for the PR!
- [PR #149](https://github.com/sendgrid/sendgrid-go/pull/149): Various typo fixes. Big thanks to [Ivan](https://github.com/janczer) for the PR!
- [PR #146](https://github.com/sendgrid/sendgrid-go/pull/146): USAGE.MD - Various grammar fixes. Big thanks to [Necroforger](https://github.com/Necroforger) for the PR!
- [PR #121](https://github.com/sendgrid/sendgrid-go/pull/121): Go lint fixes. Big thanks to [Srinivas Iyengar](https://github.com/srini156) for the PR!
- [PR #163](https://github.com/sendgrid/sendgrid-go/pull/163): Go vet fixes. Big thanks to [Vasko Zdravevski](https://github.com/vaskoz) for the PR!
- [PR #191](https://github.com/sendgrid/sendgrid-go/pull/191): Spelling corrections in md and method names. Big thanks to [Brandon Smith](https://github.com/brandon93s) for the PR!
- [PR #202](https://github.com/sendgrid/sendgrid-go/pull/202): Typos. Big thanks to [Varun Dey](https://github.com/varundey) for the PR!
- [PR #148](https://github.com/sendgrid/sendgrid-go/pull/148): Fix golint and gofmt errors. Big thanks to [Prateek Pandey](https://github.com/prateekpandey14) for the PR!
- [PR #198](https://github.com/sendgrid/sendgrid-go/pull/198): Fix wrong mail helpers example directory in README. Big thanks to [Kher Yee](https://github.com/tkbky) for the PR!
- [PR #196](https://github.com/sendgrid/sendgrid-go/pull/196): Fix for gocyclo - reducing cyclomatic complexity. Big thanks to [Srinivas Iyengar](https://github.com/srini156) for the PR!
- [PR #223](https://github.com/sendgrid/sendgrid-go/pull/223): Update LICENSE - set correct year. Big thanks to [Alex](https://github.com/pushkyn) for the PR!
- [PR #215](https://github.com/sendgrid/sendgrid-go/pull/215): Megacheck found 2 small issues. Big thanks to [Vasko Zdravevski](https://github.com/vaskoz) for the PR!
- [PR #224](https://github.com/sendgrid/sendgrid-go/pull/224): Fix spelling and formatting of comments in mail_v3.go. Big thanks to [Catlinman](https://github.com/catlinman) for the PR!
- [PR #248](https://github.com/sendgrid/sendgrid-go/pull/248): Fix license and file tests. Big thanks to [Dustin Mowcomber](https://github.com/dmowcomber) for the PR!
- [PR #252](https://github.com/sendgrid/sendgrid-go/pull/252): Add coverage.txt to .gitignore. Big thanks to [Dustin Mowcomber](https://github.com/dmowcomber) for the PR!
- [PR #261](https://github.com/sendgrid/sendgrid-go/pull/261): README tag update and linter error fix. Big thanks to [Anshul Singhal](https://github.com/af4ro) for the PR!
- [PR #273](https://github.com/sendgrid/sendgrid-go/pull/273): Exclude examples from go tests, Travis Job. Big thanks to [Fares Rihani](https://github.com/anchepiece) for the PR!
- [PR #278](https://github.com/sendgrid/sendgrid-go/pull/278): GoReportCard fixes to reach 100%. Big thanks to [Vasko Zdravevski](https://github.com/vaskoz) for the PR!
- [PR #232](https://github.com/sendgrid/sendgrid-go/pull/232): Update CONTRIBUTING.md formatting. Big thanks to [thepriefy](https://github.com/thepriefy) for the PR!
- [PR #258](https://github.com/sendgrid/sendgrid-go/pull/258): gofmt fixes. Big thanks to [ia](https://github.com/whilei) for the PR!
- [PR #292](https://github.com/sendgrid/sendgrid-go/pull/292): Fix broken link. Big thanks to [pangaunn](https://github.com/pangaunn) for the PR!
- [PR #324](https://github.com/sendgrid/sendgrid-go/pull/324): inbound: Fix readme links. Big thanks to [Arthur Silva](https://github.com/arxdsilva) for the PR!
- [PR #339](https://github.com/sendgrid/sendgrid-go/pull/339): Fix Travis builds. Big thanks to [Kevin Gillette](https://github.com/extemporalgenome) for the PR!
- [PR #321](https://github.com/sendgrid/sendgrid-go/pull/321): Clean up Dockerfile. Big thanks to [gy741](https://github.com/gy741) for the PR!

## [3.4.1] - 2017-07-03
### Added
- [Pull #116](https://github.com/sendgrid/sendgrid-go/pull/116): Fixing mimetypes in the NewSingleEmail function
- Big thanks to [Depado](https://github.com/Depado) for the pull request!

## [3.4.0] - 2017-06-14
### Added
- [Pull #96](https://github.com/sendgrid/sendgrid-go/pull/96): Send a Single Email to a Single Recipient
- Big thanks to [Oranagwa Osmond](https://github.com/andela-ooranagwa) for the pull request!

## [3.3.1] - 2016-10-18
### Fixed
- [Pull #95](https://github.com/sendgrid/sendgrid-go/pull/95): Use log instead of fmt for printing errors
- Big thanks to [Gábor Lipták](https://github.com/gliptak) for the pull request!

## [3.3.0] - 2016-10-10
### Added
- [Pull #92](https://github.com/sendgrid/sendgrid-go/pull/92): Inbound Parse Webhook support
- Checkout the [README](helpers/inbound) for details.

## [3.2.3] - 2016-10-10
### Added
- [Pull #91](https://github.com/sendgrid/sendgrid-go/pull/91): Simplified code in mail helper
- Big thanks to [Roberto Ortega](https://github.com/berto) for the pull request!

## [3.2.2] - 2016-09-08
### Added
- Merged pull request: [update prismPath and update prism binary](https://github.com/sendgrid/sendgrid-go/pull/80)
- Special thanks to [Tom Pytleski](https://github.com/pytlesk4) for the pull request!

## [3.2.1] - 2016-08-24
### Added
- Table of Contents in the README
- Added a [USE_CASES.md](USE_CASES.md) section, with the first use case example for transactional templates

## [3.2.0] - 2016-08-17
### Added
- Merged pull request: [make contents var args in NewV3MailInit](https://github.com/sendgrid/sendgrid-go/pull/75)
- The `NewV3MailInit` [Mail Helper](helpers/mail) constructor can now take in multiple content objects.
- Thanks to [Adrien Delorme](https://github.com/azr) for the pull request!

## [3.1.0] - 2016-07-28
- Dependency update to v2.2.0 of [sendGrid-rest](https://github.com/sendgrid/rest/releases/tag/v2.2.0)
- Pull [#9](https://github.com/sendgrid/rest/pull/9): Allow for setting a custom HTTP client
- [Here](https://github.com/sendgrid/rest/blob/HEAD/rest_test.go#L127) is an example of usage
- This enables usage of the [sendgrid-go library](https://github.com/sendgrid/sendgrid-go) on [Google App Engine (GAE)](https://cloud.google.com/appengine/)
- Special thanks to [Chris Broadfoot](https://github.com/broady) and [Sridhar Venkatakrishnan](https://github.com/sridharv) for providing code and feedback!

## [3.0.6] - 2016-07-26 ##
### Added
- [Troubleshooting](TROUBLESHOOTING.md) section

## [3.0.5] - 2016-07-20
### Added
- README updates
- Update introduction blurb to include information regarding our forward path
- Update the v3 /mail/send example to include non-helper usage
- Update the generic v3 example to include non-fluent interface usage

## [3.0.4] - 2016-07-12
### Added
- Update docs, unit tests and examples to include Sender ID
### Fixed
- Missing example query params for the examples

## [3.0.3] - 2016-07-08
### Fixed
- [Can't disable subscription tracking #68](https://github.com/sendgrid/sendgrid-go/issues/68)

## [3.0.2] - 2016-07-07
### Added
- Tests now mocked automatically against [prism](https://stoplight.io/prism/)

## [3.0.1] - 2016-07-05
### Added
- Accept: application/json header per https://sendgrid.com/docs/API_Reference/Web_API_v3/How_To_Use_The_Web_API_v3/requests.html

### Updated
- Content based on our updated [Swagger/OAI doc](https://github.com/sendgrid/sendgrid-oai)

## [3.0.0] - 2016-06-14
### Added
- Breaking change to support the v3 Web API
- New HTTP client
- v3 Mail Send helper

## [2.0.0] - 2015-05-02
### Changed
- Fixed a nasty bug with orphaned connections but drops support for Go versions < 1.3. Thanks [trinchan](https://github.com/sendgrid/sendgrid-go/pull/24)

## [1.2.0] - 2015-04-27
### Added
- Support for API keys

