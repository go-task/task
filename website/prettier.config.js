/**
 * @see https://prettier.io/docs/configuration
 * @type {import("prettier").Config}
 */
const config = {
  trailingComma: 'none',
  singleQuote: true,
  overrides: [
    {
      files: ['*.md'],
      options: {
        printWidth: 80,
        proseWrap: 'always'
      }
    }
  ]
};

export default config;
