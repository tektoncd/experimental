import babel from 'rollup-plugin-babel';
import cleanup from 'rollup-plugin-cleanup';
import commonjs from 'rollup-plugin-commonjs';
import externalGlobals from 'rollup-plugin-external-globals';
import postcss from 'rollup-plugin-postcss';
import replace from 'rollup-plugin-replace';
import resolve from 'rollup-plugin-node-resolve';
import svgr from '@svgr/rollup';
import url from 'rollup-plugin-url';

export default {
  input: 'src/WebhookApp.js',
  output: [
    {
      dir: 'dist',
      entryFileNames: 'extension.[hash].js',
      format: 'esm',
      //sourcemap: true
    }
  ],
  plugins: [
    resolve(),
    babel({
      plugins: [
        '@babel/plugin-proposal-object-rest-spread',
        '@babel/plugin-proposal-optional-chaining',
        '@babel/plugin-syntax-dynamic-import',
        '@babel/plugin-proposal-class-properties',
        'transform-react-remove-prop-types',
      ],
      exclude: 'node_modules/**'
    }),
    commonjs({
      include: 'node_modules/**'
    }),
    externalGlobals({
      'carbon-components-react': 'CarbonComponentsReact',
      'react': 'React',
      'react-dom': 'ReactDOM',
      'react-redux': 'ReactRedux',
      'react-router-dom': 'ReactRouterDOM'
    }),
    postcss({
      //modules: true
    }),
    replace({
      'process.env.NODE_ENV': JSON.stringify( 'production' )
    }),
    url(),
    svgr(),
    cleanup()
  ]
};