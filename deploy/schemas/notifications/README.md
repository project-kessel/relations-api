# Notifications schemas

There are three schemas presented here, demonstrating different combinations of options to model this problem.

There are two relevant dimensions to decide what schema to use:

- Should we have relationships for subscribers, unsubscribers, or both?
- Should we use relationships per event type or define event type as a caveat?

|                            | Relationships per subscriber        | Relationships per unsubscriber | Relationships for both unsubscriber & subscriber |
| -------------------------- | ----------------------------------- | ------------------------------ | ------------------------------------------------ |
| Event type in relationship | [schema_subscribers][1]             | [schema_unsubscribers][3]      | No example yet                                   |
| Event type in caveat       | [schema_subscribers_event_types][2] | No example yet                 | No example yet                                   |

[1]: ./schema_subscribers.yaml
[2]: ./schema_subscribers_event_types.yaml
[3]: ./schema_unsubscribers.yaml
