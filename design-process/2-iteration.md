# Subsequent changes from previous iteration

## Changes in plugin

- Initially to simplify things I thought to have simple plugin callbacks like HandleResponse(responsewriter) and the motivation for this simplification was inspired from how easy it is to write plugins in Openresty where each Phase gets access to a ctx on which it can operate so I plainly made a false analogy between Openresty's phases to HandleRequest() and HandleResponse() but the abstraction that openresty gives is not the same as the one net/http provides. I will have to do some very hacky way to achieve that or I will have to implement my own HTTP library to get that kind of abstraction. So I am replacing HandleResponse with WrapResponseWriter to not complicate things. :(

## No concept of middlewares

- For now I am not going to implement middlewares because I think the plugin entity I have does the job of middleware as needed.
