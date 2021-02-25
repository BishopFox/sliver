
The following function can be used to refresh the prompt, with various behavior patterns.
Please note that this function acts the same way for both 1-line and 2-line prompts, so that
you will probably need to toy around a bit with the values to attain the result you want.

```go
func(rl *Instance ) RefreshPrompt(prompt string, offset int, clearLine bool) (err error) {
        // Code
        return
}

```
The function takes into account if the shell is a 1-line/2-line prompt, so generally you should not 
consider adding related offset, or even bother thinking *Do I need to add 1, -1, or 0 to offset, 
because prompt single ?*. Don't bother with this. The examples below are by the way mixed between poth prompt styles.

### Example 1 - Refresh the prompt in place 

```go
c.shell.RefreshPromptInPlace("a new prompt that will only survive until the next readline loop")
```

![refresh-in-place](../assets/refresh-in-place.gif)


### Example 2 - Push the prompt below, a new message

As a general rule, for each line you want to print, add increment `offset` by 1. 
Setting clearLine to false keeps our current line.
```go
c.shell.RefreshPromptCustom("prompt", 0, false)
```

Note: if you use this function, it means that you have previse needs over the prompt 
refresh behavior: you will inevitably need to toy around with this to get to your ends.
The Gif below is what is produced with the call just above.

![simple-refresh](../assets/simple-refresh.gif)


### Example 3 - Push the prompt (more effiently)

This call "pushes" the prompt just below a a few lines printed. This can be used when a log line 
arrived, that you want it printed but at the same time not bothering you while working. An example:

```go
asyncLog <-logChan

// Refresh the prompt just below.
c.shell.RefreshPromptLog(asyncLog.String())
```

![refresh-prompt-log](../assets/refresh-prompt-log.gif)

