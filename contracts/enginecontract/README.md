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
  enum, range, length, and pattern validation. It has no v1 aliases and does not
  persist or expose a standalone JSON Schema projection.
