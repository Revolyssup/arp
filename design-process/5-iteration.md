# Further Optimisations

## Route Matching

Previosly matchers were being created for each Route object and were run by looping over each route. This is bad for a number of reasons. Like I can't use RadixTree or any correlation mechanism and I am having to parse the path n number of times for each route. There are all sort of problems with this approach. Now there are three types of Matchers: PathMatcher, HeaderMatcher, MethodMatcher per HTTPRouter. And as routes are added, these matcher instances update their internal state. So at the time of matching, we just have to call one matcher which can be optimised to return of all matching routes for it.
