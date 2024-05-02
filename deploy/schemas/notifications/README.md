# Notifications schemas

There are three schemas presented here, demonstrating different combinations of options to model this problem.

There are two relevant dimensions to decide what schema to use:

- Should we have relationships for subscribers, unsubscribers, or both?
- Should we use relationships per event type or define event type as a caveat?

|                            | Relationships per subscriber         | Relationships per unsubscriber | Relationships for both unsubscriber & subscriber |
| -------------------------- | ------------------------------------ | ------------------------------ | ------------------------------------------------ |
| Event type in relationship | [schema_subscribers][1]              | [schema_unsubscribers][3]      | No example yet                                   |
| Event type in caveat       | [schema_subscribers_caveat_types][2] | No example yet                 | [schema_both_caveat_types][4]                    |

[1]: ./schema_subscribers.yaml
[2]: ./schema_subscribers_caveat_types.yaml
[3]: ./schema_unsubscribers.yaml
[4]: ./schema_both_caveat_types.yaml

There are tradeoffs to consider with these different models:

- Tuples per subscriber, per event type means that for opt-out (default subscribed) notifications, we have to write _a lot_ of relations.
  Something like N event types * M total users. Additionally, for new default-subscribed events, we have to write M total user relations as part of roll out.
  We could reduce M to only _allowed_ users, but this is a much more complicated algorithm, effectively the same as AuthZed Materialize.
- Tuples per unsubscriber is the reverse. We have to write those for opt-in (default unsubscribed) notifications.
- Both allows us to choose based on the type of event, which may bring a best of both worlds. It allows us to use `user:*` for opt-out (default subscribed) relations, which is much cheaper in terms of tuples to write and subproblems to compute.
- Using a caveat for the event type reduces the number of tuples needed even further. You still have the same amount of writes, but those writes involve updating relations in place (replacing the caveat context with a different event type list), rather than adding new relations.

In short, the "relationships for both subscriber & unsubscriber" combined with event type in caveat seems to be theoretically the least work with the most flexibility for different kinds of notification defaults (default subscribed vs default unsubscribed).