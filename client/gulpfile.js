var path = require('path');
var exec = require('child_process').exec;
var url = require('url');

var gulp = require('gulp');
var gutil = require('gulp-util');
var express = require('express');
var proxy = require('express-http-proxy');
var webpack = require('webpack');
var through = require('through2');
var br = require('brotli');
var del = require('del');

function brotli(opts) {
  return through.obj(function (file, enc, callback) {
    if (file.isNull()) {
      return callback(null, file);
    }

    if (file.isStream()) {
      this.emit(
        'error',
        new gutil.PluginError('brotli', 'Streams not supported')
      );
    } else if (file.isBuffer()) {
      file.path += '.br';
      file.contents = Buffer.from(br.compress(file.contents, opts).buffer);
      return callback(null, file);
    }
  });
}

function clean() {
  return del(['dist']);
}

function js(cb) {
  var config = require('./webpack.config.prod.js');
  var compiler = webpack(config);

  process.env['NODE_ENV'] = 'production';

  compiler.run(function (err, stats) {
    if (err) throw new gutil.PluginError('webpack', err);

    gutil.log(
      '[webpack]',
      stats.toString({
        colors: true
      })
    );

    if (stats.hasErrors()) process.exit(1);

    cb();
  });
}

function config() {
  return gulp.src('../config.default.toml').pipe(gulp.dest('dist'));
}

function public() {
  return gulp.src('public/**/*').pipe(gulp.dest('dist'));
}

function compress() {
  return gulp
    .src(['dist/**/*(*.js|*.css|*.json)', '!dist/**/*(*.dev.js)'])
    .pipe(brotli({ quality: 11 }))
    .pipe(gulp.dest('dist'));
}

function cleanup() {
  return del(['dist/**/*(*.js|*.css|*.json|*.map)']);
}

function bindata(cb) {
  exec(
    'go-bindata -nomemcopy -nocompress -pkg assets -o ../assets/bindata.go -prefix "dist" dist/...',
    cb
  );
}

function serve() {
  var config = require('./webpack.config.dev.js');
  var compiler = webpack(config);
  var app = express();

  app.use(
    require('webpack-dev-middleware')(compiler, {
      noInfo: true,
      publicPath: config.output.publicPath,
      headers: {
        'Access-Control-Allow-Origin': '*'
      }
    })
  );

  app.use(require('webpack-hot-middleware')(compiler));

  app.use('/', express.static('dist'));

  app.use(
    '*',
    proxy('localhost:1337', {
      proxyReqPathResolver: function (req) {
        return req.originalUrl;
      }
    })
  );

  app.listen(3000, function (err) {
    if (err) {
      console.log(err);
      return;
    }

    console.log('Listening at http://localhost:3000');
  });
}

const build = gulp.series(
  clean,
  gulp.parallel(js, config),
  compress,
  cleanup,
  bindata
);

const dev = gulp.series(
  clean,
  gulp.parallel(serve, public, gulp.series(config, bindata))
);

gulp.task('build', build);
gulp.task('default', dev);
