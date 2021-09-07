var moment = require('moment')
var async = require('async')
var exec = require('child_process').exec
var fs = require('fs')

var isProduction = process.env.NODE_ENV == 'production' ? true : false
var assetsFileName = ''
var commitId = ''
var date = moment.utc()
var version = date.format('YYYY-MM-DDTHH:mm:ss') + 'Z'
var releaseTag = date.format('YYYY-MM-DDTHH-mm-ss') + 'Z'
var buildType = 'DEVELOPMENT'
if (process.env.IPOS_UI_BUILD) buildType = process.env.IPOS_UI_BUILD

rmDir = function(dirPath) {
  try { var files = fs.readdirSync(dirPath); }
  catch(e) { return; }
  if (files.length > 0)
    for (var i = 0; i < files.length; i++) {
      var filePath = dirPath + '/' + files[i];
      if (fs.statSync(filePath).isFile())
        fs.unlinkSync(filePath);
      else
        rmDir(filePath);
    }
  fs.rmdirSync(dirPath);
};

async.waterfall([
    function(cb) {
      rmDir('production');
      rmDir('dev');
      var cmd = 'webpack -p --config webpack.production.config.js'
      if (!isProduction) {
        cmd = 'webpack';
      }
      console.log('Running', cmd)
      exec(cmd, cb)
    },
    function(stdout, stderr, cb) {
      if (isProduction) {
        fs.renameSync('production/index_bundle.js',
                      'production/index_bundle-' + releaseTag + '.js')
      } else {
        fs.renameSync('dev/index_bundle.js',
                      'dev/index_bundle-' + releaseTag + '.js')
      }
      var cmd = 'git log --format="%H" -n1'
      console.log('Running', cmd)
      exec(cmd, cb)
    },
    function(stdout, stderr, cb) {
      if (!stdout) throw new Error('commitId is empty')
      commitId = stdout.replace('\n', '')
      if (commitId.length !== 40) throw new Error('commitId invalid : ' + commitId)
      assetsFileName = 'ui-assets.go';
      var cmd = 'go-bindata-assetfs -o bindata_assetfs.go -pkg browser -nocompress=true production/...'
      if (!isProduction) {
        cmd = 'go-bindata-assetfs -o bindata_assetfs.go -pkg browser -nocompress=true dev/...'
      }
      console.log('Running', cmd)
      exec(cmd, cb)
    },
    function(stdout, stderr, cb) {
      var cmd = 'gofmt -s -w -l bindata_assetfs.go'
      console.log('Running', cmd)
      exec(cmd, cb)
    },
    function(stdout, stderr, cb) {
      fs.renameSync('bindata_assetfs.go', assetsFileName)
      fs.appendFileSync(assetsFileName, '\n')
      fs.appendFileSync(assetsFileName, 'var UIReleaseTag = "' + buildType + '.' +
                        releaseTag + '"\n')
      fs.appendFileSync(assetsFileName, 'var UICommitID = "' + commitId + '"\n')
      fs.appendFileSync(assetsFileName, 'var UIVersion = "' + version + '"')
      fs.appendFileSync(assetsFileName, '\n')
      var contents;
      if (isProduction) {
        contents = fs.readFileSync(assetsFileName, 'utf8')
                     .replace(/_productionIndexHtml/g, '_productionIndexHTML')
                     .replace(/productionIndexHtmlBytes/g, 'productionIndexHTMLBytes')
                     .replace(/productionIndexHtml/g, 'productionIndexHTML')
                     .replace(/_productionIndex_bundleJs/g, '_productionIndexBundleJs')
                     .replace(/productionIndex_bundleJsBytes/g, 'productionIndexBundleJsBytes')
                     .replace(/productionIndex_bundleJs/g, 'productionIndexBundleJs')
                     .replace(/_productionJqueryUiMinJs/g, '_productionJqueryUIMinJs')
                     .replace(/productionJqueryUiMinJsBytes/g, 'productionJqueryUIMinJsBytes')
                     .replace(/productionJqueryUiMinJs/g, 'productionJqueryUIMinJs');
      } else {
        contents = fs.readFileSync(assetsFileName, 'utf8')
                     .replace(/_devIndexHtml/g, '_devIndexHTML')
                     .replace(/devIndexHtmlBytes/g, 'devIndexHTMLBytes')
                     .replace(/devIndexHtml/g, 'devIndexHTML')
                     .replace(/_devIndex_bundleJs/g, '_devIndexBundleJs')
                     .replace(/devIndex_bundleJsBytes/g, 'devIndexBundleJsBytes')
                     .replace(/devIndex_bundleJs/g, 'devIndexBundleJs')
                     .replace(/_devJqueryUiMinJs/g, '_devJqueryUIMinJs')
                     .replace(/devJqueryUiMinJsBytes/g, 'devJqueryUIMinJsBytes')
                     .replace(/devJqueryUiMinJs/g, 'devJqueryUIMinJs');
      }
      contents = contents.replace(/IPOS_UI_VERSION/g, version)
      contents = contents.replace(/index_bundle.js/g, 'index_bundle-' + releaseTag + '.js')

      fs.writeFileSync(assetsFileName, contents, 'utf8')
      console.log('UI assets file :', assetsFileName)
      cb()
    }
  ], function(err) {
    if (err) return console.log(err)
  })
