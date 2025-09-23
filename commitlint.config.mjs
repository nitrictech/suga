export default {
  extends: ["@commitlint/config-conventional"],
  plugins: [
    {
      rules: {
        "scope-empty-for-type": (
          parsed,
          when = "always",
          allowed = ["docs", "ci", "chore"]
        ) => {
          const { type, scope } = parsed ?? {};
          const isEmpty = !scope;
          const allowEmptyForType = allowed.includes(type);
          // When = "always": require scope for non-allowed types. "never" would disable requirement.
          const requireScope = when !== "never" && !allowEmptyForType;
          const pass = requireScope ? !isEmpty : true;
          return [
            pass,
            pass ? undefined : `scope is required for type '${type}'`,
          ];
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
      ["cli", "client", "server", "runtime", "engines", "proto", "deps"],
    ],
    "scope-empty-for-type": [2, "always", ["docs", "ci", "chore"]],
    // Disable line length restrictions
    "body-max-line-length": [0],
    "footer-max-line-length": [0],
    "header-max-length": [0],
  },
};
