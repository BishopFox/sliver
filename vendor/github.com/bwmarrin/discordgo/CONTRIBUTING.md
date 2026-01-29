# Getting started

To start off you can check out existing Pull Requests and Issues to get a gasp of what problems we’re currently solving and what features you can implement.

## Issues

Our issues are mostly used for bugs, however we welcome refactoring and conceptual issues.

Any other conversation would belong and would be moved into “Discussions”.

## Discussions

We use discussions for ideas, polls, announcements and help questions.

Don’t hesitate to ask, we always would try to help.

## Pull Requests

If you want to help us by improving existing or adding new features, you create what’s called a Pull Request (aka PR). It allows us to review your code, suggest changes and merge it.

Here are some tips on how to make a good first PR:

- When creating a PR, please consider a distinctive name and description for it, so the maintainers can understand what your PR changes / adds / removes.
- It’s always a good idea to link documentation when implementing a new feature / endpoint
- If you’re resolving an issue, don’t forget to [link it](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue) in the description.
- Enable the checkbox to allow maintainers to edit your PR and make commits in the PR branch when necessary.
- We may ask for changes, usually through suggestions or pull request comments. You can apply suggestions right in the UI. Any other change needs to be done manually.
- Don’t forget to mark PR comments resolved when you’re done applying the changes.
- Be patient and don’t close and reopen your PR when no one responds, sometimes it might be held for a while. There might be a lot of reasons: release preparation, the feature is not significant, maintainers are busy, etc.


When your changes are still incomplete (i.e. in Work In Progress state), you can still create a PR, but consider making it a draft. 
To make a draft PR, you can change the type of PR by clicking to a triangle next to the “Create Pull Request” button.

Once you’re done, you can mark it as “Ready for review”, and we’ll get right on it.


# Code style

To standardize and make things less messy we have a certain code style, that is persistent throughout the codebase.

## Naming

### REST methods

When naming a REST method, while it might seem counterintuitive, we specify the entity before the action verb (for GET endpoints we don’t specify one however). Here’s an example:

> Endpoint name: Get Channel Message
>
> Method name: `ChannelMessage`

> Endpoint name: Edit Channel Message
>
> Method name: `ChannelMessageEdit`

### Parameter structures

When making a complex REST endpoint, sometimes you might need to implement a `Param` structure. This structure contains parameters for certain endpoint/set of endpoints.

- If an endpoint/set of endpoints have mostly same parameters, it’s a good idea to use a single `Param` structure for them. Here’s an example:
    
    > Endpoint: `GuildMemberEdit`
    >
    > `Param` structure: `GuildMemberParams` 
- If an endpoint/set of endpoints have differentiating parameters, `Param` structure can be named after the endpoint’s verb. Here’s an example:
    
    > Endpoint: `ChannelMessageSendComplex`
    >
    > `Param` structure: `MessageSend`
    
    > Endpoint: `ChannelMessageEditComplex`
    >
    > `Param` structure: `MessageEdit` 

### Events

When naming an event, we follow gateway’s internal naming (which often matches with the official event name in the docs). Here’s an example:

> Event name: Interaction Create (`INTERACTION_CREATE`)
>
> Structure name: `InteractionCreate`

## Returns

In our REST functions we usually favor named returns instead of regular anonymous returns. This helps readability.

Additionally we try to avoid naked return statements for functions with a long body. Since it’s easier to loose track of the return result.
