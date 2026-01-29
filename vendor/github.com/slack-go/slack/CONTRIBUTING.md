# Contributing Guide

Welcome! We are glad that you want to contribute to our project! 💖

There are a just a few small guidelines you ask everyone to follow to make things a bit smoother and more consistent.

## Opening Pull Requests

1. It's generally best to start by opening a new issue describing the bug or feature you're intending to fix. Even if you think it's relatively minor, it's helpful to know what people are working on. Mention in the initial issue that you are planning to work on that bug or feature so that it can be assigned to you.

2. Follow the normal process of [forking](https://help.github.com/articles/fork-a-repo) the project, and set up a new branch to work in. It's important that each group of changes be done in separate branches in order to ensure that a pull request only includes the commits related to that bug or feature.

3. Any significant changes should almost always be accompanied by tests. The project already has some test coverage, so look at some of the existing tests if you're unsure how to go about it.

4. Run `make pr-prep` to format your code and check that it passes all tests and linters.

5. Do your best to have [well-formed commit messages](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html) for each change. This provides consistency throughout the project, and ensures that commit messages are able to be formatted properly by various git tools. _Pull Request Titles_ should generally follow the [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) format to ease the release note process when cutting releases.

6. Finally, push the commits to your fork and submit a [pull request](https://help.github.com/articles/creating-a-pull-request). NOTE: Please do not use force-push on PRs in this repo, as it makes it more difficult for reviewers to see what has changed since the last code review. We always perform "squash and merge" actions on PRs in this repo, so it doesn't matter how many commits your PR has, as they will end up being a single commit after merging. This is done to make a much cleaner `git log` history and helps to find regressions in the code using existing tools such as `git bisect`.

## Code Comments

Every exported method needs to have code comments that follow [Go Doc Comments](https://go.dev/doc/comment). A typical method's comments will look like this:

```go
// PostMessage sends a message to a channel.
//
// Slack API docs: https://api.dev.slack.com/methods/chat.postMessage
func (api *Client) PostMessage(ctx context.Context, input PostMesssageInput) (PostMesssageOutput, error) {
...
}
```

The first line is the name of the method followed by a short description. This could also be a longer description if needed, but there is no need to repeat any details that are documented in Slack's documentation because users are expected to follow the documentation links to learn more.

After the description comes a link to the Slack API documentation.

## Other notes on code organization

Currently, everything is defined in the main `slack` package, with API methods group separate files by the [Slack API Method Groupings](https://api.dev.slack.com/methods).
