# How To Contribute to Twilio SendGrid Repositories via GitHub
Contributing to the Twilio SendGrid repositories is easy! All you need to do is find an open issue (see the bottom of this page for a list of repositories containing open issues), fix it and submit a pull request. Once you have submitted your pull request, the team can easily review it before it is merged into the repository.

To make a pull request, follow these steps:

1. Log into GitHub. If you do not already have a GitHub account, you will have to create one in order to submit a change. Click the Sign up link in the upper right-hand corner to create an account. Enter your username, password, and email address. If you are an employee of Twilio SendGrid, please use your full name with your GitHub account and enter Twilio SendGrid as your company so we can easily identify you.

<img src="/static/img/github-sign-up.png" width="800">

2. __[Fork](https://help.github.com/fork-a-repo/)__ the [sendgrid-go](https://github.com/sendgrid/sendgrid-go) repository:

<img src="/static/img/github-fork.png" width="800">

3. __Clone__  your fork via the following commands:

```bash
# Clone your fork of the repo into the current directory
git clone https://github.com/your_username/sendgrid-go
# Navigate to the newly cloned directory
cd sendgrid-go
# Assign the original repo to a remote called "upstream"
git remote add upstream https://github.com/sendgrid/sendgrid-go
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
