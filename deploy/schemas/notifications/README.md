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

## Use cases

### Querying for users to notify

Pre-requisite work:

- There must be some function F(event) which results in:
   - A reference to a specific object type and ID relevant to that event (e.g. `inventory/host:1`)
   - An access related event type code or name (e.g. `view` or `new_cve`). This depends on how granular the access control is for that event.

When an event occurs which may result in a notification...

1. Apply F(event) to get the object and event type
2. Run a `lookup-subjects` query to find the users to notify, using the output of step 1 as the inputs to the query. E.g. `lookup-subjects <object> <event_type>_notification user` or `lookup-subjects inventory/host:1 view_notification user`
3. As a result you'll get a stream of user IDs. The rest of this pipeline is an exercise for the reader, but for example you could publish these IDs to another async part of the processing flow which could lookup that user's delivery preferences like time or email address and batch or send the notifications accordingly.
