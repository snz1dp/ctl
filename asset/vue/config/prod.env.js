'use strict'

const path = require('path')
const fs = require('fs')
const packageJson = JSON.parse(fs.readFileSync(path.join(__dirname, '../', 'package.json')))

module.exports = {
  NODE_ENV: '"production"',
  // TODO: 修改前端地址
  BASE_URL: '"{{ .BasePath }}"',
  // TODO: 修改后端地址
  BASE_API: '"{{ .ApiPath }}"',
  XEAI_API: '"/xeai"',
  APP_CODE: '"' + packageJson.name + '"',
  APP_VERSION: '"' + packageJson.version + '"',
  APP_TITLE: '"' + packageJson.description + '"',
}
