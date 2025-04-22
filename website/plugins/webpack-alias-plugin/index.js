const path = require('path');

module.exports = function webpackAliasPlugin(context, options) {
  return {
    name: 'webpack-alias-plugin',
    configureWebpack(config, isServer, utils) {
      return {
        resolve: {
          alias: {
            '~': path.resolve(__dirname, '../../src'),
          },
        },
      };
    },
  };
};
