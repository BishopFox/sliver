# How To Contribute to Twilio SendGrid Repositories via GitHub
Contributing to the Twilio SendGrid repositories is easy! All you need to do is find an open issue (see the bottom of this page for a list of repositories containing open issues), fix it and submit a pull request. Once you have submitted your pull request, the team can easily review it before it is merged into the repository.

To make a pull request, follow these steps:

1. Log into GitHub. If you do not already have a GitHub account, you will have to create one in order to submit a change. Click the Sign up link in the upper right-hand corner to create an account. Enter your username, password, and email address. If you are an employee of Twilio SendGrid, please use your full name with your GitHub account and enter Twilio SendGrid as your company so we can easily identify you.

<img src="/static/img/github-sign-up.png" width="800">

2. __[Fork](https://help.github.com/fork-a-repo/)__ the [rest](https://github.com/sendgrid/rest) repository:

<img src="/static/img/github-fork.png" width="800">

3. __Clone__  your fork via the following commands:

```bash
# Clone your fork of the repo into the current directory
git clone https://github.com/your_username/rest
# Navigate to the newly cloned directory
cd rest
# Assign the original repo to a remote called "upstream"
git remote add upstream https://github.com/sendgrid/rest
```

> Don't forget to replace *your_username* in the URL by your real GitHub username.

4. __Create a new topic branch__ (off the main project development branch) to contain your feature, change, or fix:

```bash
git checkout -b <topic-branch-name>
```

5. __Commit your changes__ in logical chunks.

Please adhere to these [git commit message guidelines](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html) or your code is unlikely be merged into the main project. Use Git's [interactive rebase](https://help.github.com/articles/interactive-rebase) feature to tidy up your commits before making them public. Probably you will also have to create tests (if needed) or create or update the example code that demonstrates the functionality of this change to the code.

6. __Locally merge (or rebase)__ the upstream development branch into your topic branch:

```bash
git pull [--rebase] upstream main
```

7. __Push__ your topic branch up to your fork:

```bash
git push origin <topic-branch-name>
```

8. __[Open a Pull Request](https://help.github.com/articles/creating-a-pull-request/#changing-the-branch-range-and-destination-repository/)__ with a clear title and description against the `main` branch. All tests must be passing before we will review the PR.

## Important notice

Before creating a pull request, make sure that you respect the repository's constraints regarding contributions. You can find them in the [CONTRIBUTING.md](CONTRIBUTING.md) file.

## Repositories with Open, Easy, Help Wanted, Issue Filters

* [Python SDK](https://github.com/sendgrid/sendgrid-python/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [PHP SDK](https://github.com/sendgrid/sendgrid-php/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [C# SDK](https://github.com/sendgrid/sendgrid-csharp/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Ruby SDK](https://github.com/sendgrid/sendgrid-ruby/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Node.js SDK](https://github.com/sendgrid/sendgrid-nodejs/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Java SDK](https://github.com/sendgrid/sendgrid-java/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Go SDK](https://github.com/sendgrid/sendgrid-go/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Python SMTPAPI Client](https://github.com/sendgrid/smtpapi-python/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [PHP SMTPAPI Client](https://github.com/sendgrid/smtpapi-php/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [C# SMTPAPI Client](https://github.com/sendgrid/smtpapi-csharp/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Ruby SMTPAPI Client](https://github.com/sendgrid/smtpapi-ruby/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Node.js SMTPAPI Client](https://github.com/sendgrid/smtpapi-nodejs/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Java SMTPAPI Client](https://github.com/sendgrid/smtpapi-java/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Go SMTPAPI Client](https://github.com/sendgrid/smtpapi-go/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Python HTTP Client](https://github.com/sendgrid/python-http-client/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [PHP HTTP Client](https://github.com/sendgrid/php-http-client/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [C# HTTP Client](https://github.com/sendgrid/csharp-http-client/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Java HTTP Client](https://github.com/sendgrid/java-http-client/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Ruby HTTP Client](https://github.com/sendgrid/ruby-http-client/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Go HTTP Client](https://github.com/sendgrid/rest/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Open API Definition](https://github.com/sendgrid/sendgrid-oai/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [DX Automator](https://github.com/sendgrid/dx-automator/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
* [Documentation](https://github.com/sendgrid/docs/issues?utf8=%E2%9C%93&q=is%3Aopen+label%3A%22difficulty%3A+easy%22+label%3A%22status%3A+help+wanted%22)
