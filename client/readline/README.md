# Readline - Enhanced version of [readline](https://github.com/lmorg/readline) console

The original project by lmorg is absolutely great. The console already has superb capabilities,
and many thanks go to its author: it was both simple and fast to add the various enhancements:

- **Better completion system** *à-la-ZSH* offering multiple categories of completions, each with their
   own display types, concurrently. Also works in completion search mode.
- **Better Vim mode indicators**. The shell can now (optionally) give a clear and real-time 
   indication of the current Vim Mode. 
- **Prompt multiline**. You can either have a single line prompt, or a two-line prompt *à-la-Spaceship-prompt*. 
- **Prompt refresh**. A new function is added, and can be called for live refresh of the prompt string part (full
   line refresh is already implicitely proposed with the SyntaxCompleter() function)


## TODO 

- A simple function for refreshing the prompt part of the input line. 
    instance.RefreshPrompt([]rune{})
- When proposing completions, analyze the input line and find items that are already input but still in completions.
  Then, recompute completions to delete these items, like in ZSH.
