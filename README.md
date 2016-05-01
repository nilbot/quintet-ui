# quintet ui
quintet frontend programmed in Go

## Why
It's 2016, shouldn't be using Java to code quintet in the first place...

## What
This ui is primarily a fancy display and you can watch it from any device, provided you use modern browser (>IE9)
 - Direct: open the browser and the information will come to you, in real time.
 - Performance: hundreds, thousands client can watch the result simultaneously, hop on hop off, no loading, no queuing.
 - Robust: operators may go, watchers may go, the server will simply collect the garbage and will not crash.
 - Visual: fancy graphs are generated as data streamed in. [see https://github.com/nilbot/chart]
 
## TODO
Here is the list
- [ ] User assume control as operator, this means remote execution and requires some form of authentication. But if this tool suite is deployed in controlled environment, we can spare the authentication.
- [ ] Data hotdrop: config, input data should be able to uploaded by hot drop.

## How
### Observe
https://demo.nilbot.net
### Operate
1. obtain quintet package
2. `java -jar quintet-1.2.0.jar`
3. modify `resources/config.properties` for more options
 


