# Changelog

## [0.1.1](https://github.com/goatx/goat/releases/tag/v0.1.1) - 2025-08-16

### Added
- Multi state machine condition support
  - `NewMultiCondition` function to create conditions that reference multiple state machines
  - `NewCondition2` and `NewCondition3` convenience functions for 2 or 3 state machines
- Condition-based invariant registration
  - `WithConditions` to register named predicates
  - `WithInvariants` to mark conditions for global checking
- Temporal rule checking with `WithTemporalRules`
  - `WheneverPEventuallyQ`, `EventuallyAlways`, and `AlwaysEventually` helpers

## [0.1.0](https://github.com/goatx/goat/releases/tag/v0.1.0) - 2025-07-31

- Initial release of goat
