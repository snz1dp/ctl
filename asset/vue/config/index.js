'use strict'
// Template version: 1.3.1
// see http://vuejs-templates.github.io/webpack for documentation.

const path = require('path')

// TODO: 修改前端地址
const assetsPublicPath = '{{ .BasePath }}/'

module.exports = {
  dev: {

    // Paths
    assetsSubDirectory: 'static',
    assetsPublicPath: assetsPublicPath,

    // 跨域访问API代理配置
    proxyTable: {
      '{{ .ApiPath }}': {
        target: '{{ .ApiBase }}',
        changeOrigin: true,
        pathRewrite: {
          '^{{ .ApiPath }}': '{{ .ApiPath }}'
        },
        proxyTimeout: 18000000
      },
      '/xeai': {
        target: 'http://localhost:8585',
        changeOrigin: true,
        pathRewrite: {
          '^/xeai': '/xeai'
        }
      }
    },

    // TODO: 修改监听地址
    host: '0.0.0.0', // can be overwritten by process.env.HOST
    // TODO: 修改监听端口
    port: 9301, // can be overwritten by process.env.PORT, if port is in use, a free one will be determined

    autoOpenBrowser: false,
    errorOverlay: true,
    notifyOnErrors: true,
    poll: false, // https://webpack.js.org/configuration/dev-server/#devserver-watchoptions-

    // Use Eslint Loader?
    // If true, your code will be linted during bundling and
    // linting errors and warnings will be shown in the console.
    useEslint: true,
    // If true, eslint errors and warnings will also be shown in the error overlay
    // in the browser.
    showEslintErrorsInOverlay: false,

    /**
     * Source Maps
     */

    // https://webpack.js.org/configuration/devtool/#development
    devtool: 'cheap-module-eval-source-map',

    // If you have problems debugging vue-files in devtools,
    // set this to false - it *may* help
    // https://vue-loader.vuejs.org/en/options.html#cachebusting
    cacheBusting: true,

    cssSourceMap: true
  },

  build: {
    // Template for index.html
    index: path.resolve(__dirname, '../dist/index.html'),

    // Paths
    assetsRoot: path.resolve(__dirname, '../dist'),
    assetsSubDirectory: 'static',
    assetsPublicPath: assetsPublicPath,

    /**
     * Source Maps
     */

    productionSourceMap: true,

    // https://webpack.js.org/configuration/devtool/#production
    devtool: '#source-map',

    // Gzip off by default as many popular static hosts such as
    // Surge or Netlify already gzip all static assets for you.
    // Before setting to `true`, make sure to:
    // npm install --save-dev compression-webpack-plugin
    productionGzip: false,
    productionGzipExtensions: ['js', 'css'],

    // Run the build command with an extra argument to
    // View the bundle analyzer report after build finishes:
    // `npm run build --report`
    // Set to `true` or `false` to always turn it on or off
    bundleAnalyzerReport: process.env.npm_config_report
  }
}
