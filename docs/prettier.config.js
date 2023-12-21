module.exports = {
  trailingComma: 'none',
  singleQuote: true,
  overrides: [
    {
      files: ['*.md', '*.mdx'],
      options: {
        printWidth: 80,
        proseWrap: 'always'
      }
    }
  ]
};
