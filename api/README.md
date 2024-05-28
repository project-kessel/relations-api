# API Design

API design should refer to the following style guides, in this order of precedence:

- https://protobuf.dev/programming-guides/proto3/
- https://protobuf.dev/programming-guides/dos-donts/
- https://protobuf.dev/programming-guides/api
- https://buf.build/docs/best-practices/style-guide
- https://cloud.google.com/apis/design

## Notes

These are some specific call outs from the above docs which are useful or surprising in case you
didn't spend the required time to go through all of those links.

### Service names

Service names should be globally disambiguated, even regardless of package, hence the inclusion of
"Kessel" in the names (https://protobuf.dev/programming-guides/api/#service-name-globally-unique).

### Comments

Comment fields using Markdown (https://buf.build/docs/bsr/documentation#supported-formatting-syntax).

### Method names

These generally follow Google Cloud's guidance, with VerbNoun, in imperative mood, with a standard
set of verbs. Custom method names are allowed outside of this.

- https://cloud.google.com/apis/design/naming_convention#method_names
- https://cloud.google.com/apis/design/standard_methods

Note that "List" uses paged messages, _not_ streams,
so "List" should be avoided for methods with streaming responses.

It does not appear there is any other convention for methods with streaming responses,
[based on querying all of GCP's APIs][1].

[1]: https://github.com/search?q=repo:googleapis/googleapis+%22returns+(stream+%22+language:%22Protocol+Buffer%22&type=code&p=1

## Troubleshooting

### Using VSCode proto extension and seeing import errors?

Add this to your settings.json:

```
   "protoc": {
    "options": ["--proto_path=api", "--proto_path=third_party"]
  }
```
