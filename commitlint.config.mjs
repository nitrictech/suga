export default {
  extends: ["@commitlint/config-conventional"],
  plugins: [
    {
      rules: {
        "scope-empty-for-type": ({ type, scope }) => {
          // Allow empty scope for docs, ci, and chore commits
          const typesAllowingEmptyScope = ["docs", "ci", "chore"];
          if (typesAllowingEmptyScope.includes(type) && !scope) {
            return [true];
          }
          // Require scope for other types
          if (!typesAllowingEmptyScope.includes(type) && !scope) {
            return [false, `scope is required for type '${type}'`];
          }
          return [true];
        },
      },
    },
  ],
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "feat",
        "fix",
        "docs",
        "refactor",
        "perf",
        "test",
        "ci",
        "chore",
        "revert",
      ],
    ],
    "scope-enum": [
      2,
      "always",
      [
        "cli",
        "client",
        "server",
        "runtime",
        "engines",
        "proto",
        "docs",
        "deps",
        "ci",
        "release",
      ],
    ],
    "scope-empty-for-type": [2, "always"],
  },
};
