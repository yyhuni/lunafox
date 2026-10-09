# Engine Contracts

`contracts/enginecontract` contains stable engine authoring contracts shared by
Server package consumers and repository tooling.

Packages under this tree define engine declarations and Definition-backed config
defaulting and validation. Agent consumes only the compiled execution plan and
must not import or reinterpret these authoring contracts. Do not place
engine-process SDK entrypoints, host launch mechanics, transport sessions,
package loaders, build helpers, or compatibility aliases here.

Current packages:

- `engineexecution`: the versioned `ExecutionDefinition` model, strict
  normalized decoding/cloning, and config defaulting plus closed-shape, type,
  enum, range, length, pattern, and semantic format validation. It has no v1 aliases and does not
  persist or expose a standalone JSON Schema projection.

Parameter `format` is optional and currently accepts only `http-headers` on a
`stringArray` without `enum`. Each value must be a `Name: Value` HTTP request
header with a valid token name and non-empty valid value; CR/LF and invalid
controls fail before execution. Defaults and runtime values use the same HTTP
grammar. Empty arrays are valid, and validation preserves bytes and order.
Errors identify the field/index without exposing header contents. Format is
semantic metadata, independent of the parameter key; it is not a UI widget or
a storage/log confidentiality annotation.
