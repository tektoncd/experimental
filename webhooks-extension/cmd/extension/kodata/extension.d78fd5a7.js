function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) {
  try {
    var info = gen[key](arg);
    var value = info.value;
  } catch (error) {
    reject(error);
    return;
  }

  if (info.done) {
    resolve(value);
  } else {
    Promise.resolve(value).then(_next, _throw);
  }
}

function _asyncToGenerator(fn) {
  return function () {
    var self = this,
        args = arguments;
    return new Promise(function (resolve, reject) {
      var gen = fn.apply(self, args);

      function _next(value) {
        asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value);
      }

      function _throw(err) {
        asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err);
      }

      _next(undefined);
    });
  };
}

function _classCallCheck(instance, Constructor) {
  if (!(instance instanceof Constructor)) {
    throw new TypeError("Cannot call a class as a function");
  }
}

function _defineProperties(target, props) {
  for (var i = 0; i < props.length; i++) {
    var descriptor = props[i];
    descriptor.enumerable = descriptor.enumerable || false;
    descriptor.configurable = true;
    if ("value" in descriptor) descriptor.writable = true;
    Object.defineProperty(target, descriptor.key, descriptor);
  }
}

function _createClass(Constructor, protoProps, staticProps) {
  if (protoProps) _defineProperties(Constructor.prototype, protoProps);
  if (staticProps) _defineProperties(Constructor, staticProps);
  return Constructor;
}

function _defineProperty(obj, key, value) {
  if (key in obj) {
    Object.defineProperty(obj, key, {
      value: value,
      enumerable: true,
      configurable: true,
      writable: true
    });
  } else {
    obj[key] = value;
  }

  return obj;
}

function _extends() {
  _extends = Object.assign || function (target) {
    for (var i = 1; i < arguments.length; i++) {
      var source = arguments[i];

      for (var key in source) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
          target[key] = source[key];
        }
      }
    }

    return target;
  };

  return _extends.apply(this, arguments);
}

function _objectSpread(target) {
  for (var i = 1; i < arguments.length; i++) {
    var source = arguments[i] != null ? arguments[i] : {};
    var ownKeys = Object.keys(source);

    if (typeof Object.getOwnPropertySymbols === 'function') {
      ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) {
        return Object.getOwnPropertyDescriptor(source, sym).enumerable;
      }));
    }

    ownKeys.forEach(function (key) {
      _defineProperty(target, key, source[key]);
    });
  }

  return target;
}

function _inherits(subClass, superClass) {
  if (typeof superClass !== "function" && superClass !== null) {
    throw new TypeError("Super expression must either be null or a function");
  }

  subClass.prototype = Object.create(superClass && superClass.prototype, {
    constructor: {
      value: subClass,
      writable: true,
      configurable: true
    }
  });
  if (superClass) _setPrototypeOf(subClass, superClass);
}

function _getPrototypeOf(o) {
  _getPrototypeOf = Object.setPrototypeOf ? Object.getPrototypeOf : function _getPrototypeOf(o) {
    return o.__proto__ || Object.getPrototypeOf(o);
  };
  return _getPrototypeOf(o);
}

function _setPrototypeOf(o, p) {
  _setPrototypeOf = Object.setPrototypeOf || function _setPrototypeOf(o, p) {
    o.__proto__ = p;
    return o;
  };

  return _setPrototypeOf(o, p);
}

function _assertThisInitialized(self) {
  if (self === void 0) {
    throw new ReferenceError("this hasn't been initialised - super() hasn't been called");
  }

  return self;
}

function _possibleConstructorReturn(self, call) {
  if (call && (typeof call === "object" || typeof call === "function")) {
    return call;
  }

  return _assertThisInitialized(self);
}

var defaultOptions = {
  method: 'get'
};
function getHeaders() {
  var headers = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : {};
  return _objectSpread({
    Accept: 'application/json',
    'Content-Type': 'application/json'
  }, headers);
}
function checkStatus() {
  var response = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : {};
  if (response.ok) {
    switch (response.status) {
      case 201:
        return response.headers;
      case 204:
        return {};
      default:
        return response.json();
    }
  }
  var error = new Error(response.statusText);
  error.response = response;
  throw error;
}
function request(uri) {
  var options = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : defaultOptions;
  return fetch(uri, _objectSpread({}, options)).then(checkStatus);
}
function get(uri) {
  return request(uri, {
    method: 'get',
    headers: getHeaders()
  });
}
function post(uri, body) {
  return request(uri, {
    method: 'post',
    headers: getHeaders(),
    body: JSON.stringify(body)
  });
}
function deleteRequest(uri) {
  return request(uri, {
    method: 'delete',
    headers: getHeaders()
  });
}

var apiRoot = getAPIRoot();
var dashboardAPIRoot = getDashboardAPIRoot();
function getAPIRoot() {
  var _window$location = window.location,
      href = _window$location.href,
      hash = _window$location.hash;
  var newHash = hash.replace('#/extensions', 'v1/extensions');
  var baseURL = href.replace(hash, newHash);
  if (baseURL.endsWith('/')) {
    baseURL = baseURL.slice(0, -1);
  }
  return baseURL;
}
function getDashboardAPIRoot() {
  var _window$location2 = window.location,
      href = _window$location2.href,
      hash = _window$location2.hash;
  var newHash = hash.replace('#/extensions/webhooks-extension', 'v1');
  var baseURL = href.replace(hash, newHash);
  if (baseURL.endsWith('/')) {
    baseURL = baseURL.slice(0, -1);
  }
  return baseURL;
}
function getWebhooks() {
  var uri = "".concat(apiRoot, "/webhooks");
  return get(uri);
}
function createWebhook(data) {
  var uri = "".concat(apiRoot, "/webhooks");
  return post(uri, data);
}
function getSecrets(namespace) {
  var uri = "".concat(apiRoot, "/webhooks/credentials?namespace=").concat(namespace);
  return get(uri);
}
function createSecret(data, namespace) {
  var uri = "".concat(apiRoot, "/webhooks/credentials?namespace=").concat(namespace);
  return post(uri, data);
}
function deleteSecret(name, namespace) {
  var uri = "".concat(apiRoot, "/webhooks/credentials/").concat(name, "?namespace=").concat(namespace);
  return deleteRequest(uri);
}
function getNamespaces() {
  var uri = "".concat(dashboardAPIRoot, "/namespaces");
  return get(uri);
}
function getPipelines(namespace) {
  var uri = "".concat(dashboardAPIRoot, "/namespaces/").concat(namespace, "/pipelines");
  return get(uri);
}
function getServiceAccounts(namespace) {
  var uri = "".concat(dashboardAPIRoot, "/namespaces/").concat(namespace, "/serviceaccounts");
  return get(uri);
}

function _defineProperty$1(obj, key, value) {
  if (key in obj) {
    Object.defineProperty(obj, key, {
      value: value,
      enumerable: true,
      configurable: true,
      writable: true
    });
  } else {
    obj[key] = value;
  }
  return obj;
}
function _objectSpread$1(target) {
  for (var i = 1; i < arguments.length; i++) {
    var source = arguments[i] != null ? arguments[i] : {};
    var ownKeys = Object.keys(source);
    if (typeof Object.getOwnPropertySymbols === 'function') {
      ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) {
        return Object.getOwnPropertyDescriptor(source, sym).enumerable;
      }));
    }
    ownKeys.forEach(function (key) {
      _defineProperty$1(target, key, source[key]);
    });
  }
  return target;
}
function _objectWithoutPropertiesLoose(source, excluded) {
  if (source == null) return {};
  var target = {};
  var sourceKeys = Object.keys(source);
  var key, i;
  for (i = 0; i < sourceKeys.length; i++) {
    key = sourceKeys[i];
    if (excluded.indexOf(key) >= 0) continue;
    target[key] = source[key];
  }
  return target;
}
function _objectWithoutProperties(source, excluded) {
  if (source == null) return {};
  var target = _objectWithoutPropertiesLoose(source, excluded);
  var key, i;
  if (Object.getOwnPropertySymbols) {
    var sourceSymbolKeys = Object.getOwnPropertySymbols(source);
    for (i = 0; i < sourceSymbolKeys.length; i++) {
      key = sourceSymbolKeys[i];
      if (excluded.indexOf(key) >= 0) continue;
      if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue;
      target[key] = source[key];
    }
  }
  return target;
}
var defaultAttributes = {
  focusable: 'false',
  preserveAspectRatio: 'xMidYMid meet',
  style: 'will-change: transform;'
};
function getAttributes() {
  var _ref = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : {},
      width = _ref.width,
      height = _ref.height,
      _ref$viewBox = _ref.viewBox,
      viewBox = _ref$viewBox === void 0 ? "0 0 ".concat(width, " ").concat(height) : _ref$viewBox,
      attributes = _objectWithoutProperties(_ref, ["width", "height", "viewBox"]);
  var tabindex = attributes.tabindex,
      rest = _objectWithoutProperties(attributes, ["tabindex"]);
  var iconAttributes = _objectSpread$1({}, defaultAttributes, rest, {
    width: width,
    height: height,
    viewBox: viewBox
  });
  if (iconAttributes['aria-label'] || iconAttributes['aria-labelledby'] || iconAttributes.title) {
    iconAttributes.role = 'img';
    if (tabindex !== undefined && tabindex !== null) {
      iconAttributes.focusable = 'true';
      iconAttributes.tabindex = tabindex;
    }
  } else {
    iconAttributes['aria-hidden'] = true;
  }
  return iconAttributes;
}
function toString(descriptor) {
  var _descriptor$elem = descriptor.elem,
      elem = _descriptor$elem === void 0 ? 'svg' : _descriptor$elem,
      _descriptor$attrs = descriptor.attrs,
      attrs = _descriptor$attrs === void 0 ? {} : _descriptor$attrs,
      _descriptor$content = descriptor.content,
      content = _descriptor$content === void 0 ? [] : _descriptor$content;
  var children = content.map(toString).join('');
  if (elem !== 'svg') {
    return "<".concat(elem, " ").concat(formatAttributes(attrs), ">").concat(children, "</").concat(elem, ">");
  }
  return "<".concat(elem, " ").concat(formatAttributes(getAttributes(attrs)), ">").concat(children, "</").concat(elem, ">");
}
function formatAttributes(attrs) {
  return Object.keys(attrs).reduce(function (acc, key, index) {
    var attribute = "".concat(key, "=\"").concat(attrs[key], "\"");
    if (index === 0) {
      return attribute;
    }
    return acc + ' ' + attribute;
  }, '');
}
function toSVG(descriptor) {
  var _descriptor$elem = descriptor.elem,
      elem = _descriptor$elem === void 0 ? 'svg' : _descriptor$elem,
      _descriptor$attrs = descriptor.attrs,
      attrs = _descriptor$attrs === void 0 ? {} : _descriptor$attrs,
      _descriptor$content = descriptor.content,
      content = _descriptor$content === void 0 ? [] : _descriptor$content;
  var node = document.createElementNS('http://www.w3.org/2000/svg', elem);
  var attributes = elem !== 'svg' ? attrs : getAttributes(attrs);
  Object.keys(attributes).forEach(function (key) {
    node.setAttribute(key, attrs[key]);
  });
  for (var i = 0; i < content.length; i++) {
    node.appendChild(toSVG(content[i]));
  }
  return node;
}

var es = /*#__PURE__*/Object.freeze({
  defaultAttributes: defaultAttributes,
  getAttributes: getAttributes,
  formatAttributes: formatAttributes,
  toString: toString,
  toSVG: toSVG
});

function unwrapExports (x) {
	return x && x.__esModule && Object.prototype.hasOwnProperty.call(x, 'default') ? x['default'] : x;
}

function createCommonjsModule(fn, module) {
	return module = { exports: {} }, fn(module, module.exports), module.exports;
}

var reactIs_production_min = createCommonjsModule(function (module, exports) {
Object.defineProperty(exports,"__esModule",{value:!0});
var b="function"===typeof Symbol&&Symbol.for,c=b?Symbol.for("react.element"):60103,d=b?Symbol.for("react.portal"):60106,e=b?Symbol.for("react.fragment"):60107,f=b?Symbol.for("react.strict_mode"):60108,g=b?Symbol.for("react.profiler"):60114,h=b?Symbol.for("react.provider"):60109,k=b?Symbol.for("react.context"):60110,l=b?Symbol.for("react.async_mode"):60111,m=b?Symbol.for("react.concurrent_mode"):60111,n=b?Symbol.for("react.forward_ref"):60112,p=b?Symbol.for("react.suspense"):60113,q=b?Symbol.for("react.memo"):
60115,r=b?Symbol.for("react.lazy"):60116;function t(a){if("object"===typeof a&&null!==a){var u=a.$$typeof;switch(u){case c:switch(a=a.type,a){case l:case m:case e:case g:case f:case p:return a;default:switch(a=a&&a.$$typeof,a){case k:case n:case h:return a;default:return u}}case r:case q:case d:return u}}}function v(a){return t(a)===m}exports.typeOf=t;exports.AsyncMode=l;exports.ConcurrentMode=m;exports.ContextConsumer=k;exports.ContextProvider=h;exports.Element=c;exports.ForwardRef=n;
exports.Fragment=e;exports.Lazy=r;exports.Memo=q;exports.Portal=d;exports.Profiler=g;exports.StrictMode=f;exports.Suspense=p;exports.isValidElementType=function(a){return "string"===typeof a||"function"===typeof a||a===e||a===m||a===g||a===f||a===p||"object"===typeof a&&null!==a&&(a.$$typeof===r||a.$$typeof===q||a.$$typeof===h||a.$$typeof===k||a.$$typeof===n)};exports.isAsyncMode=function(a){return v(a)||t(a)===l};exports.isConcurrentMode=v;exports.isContextConsumer=function(a){return t(a)===k};
exports.isContextProvider=function(a){return t(a)===h};exports.isElement=function(a){return "object"===typeof a&&null!==a&&a.$$typeof===c};exports.isForwardRef=function(a){return t(a)===n};exports.isFragment=function(a){return t(a)===e};exports.isLazy=function(a){return t(a)===r};exports.isMemo=function(a){return t(a)===q};exports.isPortal=function(a){return t(a)===d};exports.isProfiler=function(a){return t(a)===g};exports.isStrictMode=function(a){return t(a)===f};
exports.isSuspense=function(a){return t(a)===p};
});
unwrapExports(reactIs_production_min);
var reactIs_production_min_1 = reactIs_production_min.typeOf;
var reactIs_production_min_2 = reactIs_production_min.AsyncMode;
var reactIs_production_min_3 = reactIs_production_min.ConcurrentMode;
var reactIs_production_min_4 = reactIs_production_min.ContextConsumer;
var reactIs_production_min_5 = reactIs_production_min.ContextProvider;
var reactIs_production_min_6 = reactIs_production_min.Element;
var reactIs_production_min_7 = reactIs_production_min.ForwardRef;
var reactIs_production_min_8 = reactIs_production_min.Fragment;
var reactIs_production_min_9 = reactIs_production_min.Lazy;
var reactIs_production_min_10 = reactIs_production_min.Memo;
var reactIs_production_min_11 = reactIs_production_min.Portal;
var reactIs_production_min_12 = reactIs_production_min.Profiler;
var reactIs_production_min_13 = reactIs_production_min.StrictMode;
var reactIs_production_min_14 = reactIs_production_min.Suspense;
var reactIs_production_min_15 = reactIs_production_min.isValidElementType;
var reactIs_production_min_16 = reactIs_production_min.isAsyncMode;
var reactIs_production_min_17 = reactIs_production_min.isConcurrentMode;
var reactIs_production_min_18 = reactIs_production_min.isContextConsumer;
var reactIs_production_min_19 = reactIs_production_min.isContextProvider;
var reactIs_production_min_20 = reactIs_production_min.isElement;
var reactIs_production_min_21 = reactIs_production_min.isForwardRef;
var reactIs_production_min_22 = reactIs_production_min.isFragment;
var reactIs_production_min_23 = reactIs_production_min.isLazy;
var reactIs_production_min_24 = reactIs_production_min.isMemo;
var reactIs_production_min_25 = reactIs_production_min.isPortal;
var reactIs_production_min_26 = reactIs_production_min.isProfiler;
var reactIs_production_min_27 = reactIs_production_min.isStrictMode;
var reactIs_production_min_28 = reactIs_production_min.isSuspense;

var reactIs_development = createCommonjsModule(function (module, exports) {
});
unwrapExports(reactIs_development);
var reactIs_development_1 = reactIs_development.typeOf;
var reactIs_development_2 = reactIs_development.AsyncMode;
var reactIs_development_3 = reactIs_development.ConcurrentMode;
var reactIs_development_4 = reactIs_development.ContextConsumer;
var reactIs_development_5 = reactIs_development.ContextProvider;
var reactIs_development_6 = reactIs_development.Element;
var reactIs_development_7 = reactIs_development.ForwardRef;
var reactIs_development_8 = reactIs_development.Fragment;
var reactIs_development_9 = reactIs_development.Lazy;
var reactIs_development_10 = reactIs_development.Memo;
var reactIs_development_11 = reactIs_development.Portal;
var reactIs_development_12 = reactIs_development.Profiler;
var reactIs_development_13 = reactIs_development.StrictMode;
var reactIs_development_14 = reactIs_development.Suspense;
var reactIs_development_15 = reactIs_development.isValidElementType;
var reactIs_development_16 = reactIs_development.isAsyncMode;
var reactIs_development_17 = reactIs_development.isConcurrentMode;
var reactIs_development_18 = reactIs_development.isContextConsumer;
var reactIs_development_19 = reactIs_development.isContextProvider;
var reactIs_development_20 = reactIs_development.isElement;
var reactIs_development_21 = reactIs_development.isForwardRef;
var reactIs_development_22 = reactIs_development.isFragment;
var reactIs_development_23 = reactIs_development.isLazy;
var reactIs_development_24 = reactIs_development.isMemo;
var reactIs_development_25 = reactIs_development.isPortal;
var reactIs_development_26 = reactIs_development.isProfiler;
var reactIs_development_27 = reactIs_development.isStrictMode;
var reactIs_development_28 = reactIs_development.isSuspense;

var reactIs = createCommonjsModule(function (module) {
{
  module.exports = reactIs_production_min;
}
});

/*
object-assign
(c) Sindre Sorhus
@license MIT
*/
var getOwnPropertySymbols = Object.getOwnPropertySymbols;
var hasOwnProperty = Object.prototype.hasOwnProperty;
var propIsEnumerable = Object.prototype.propertyIsEnumerable;
function toObject(val) {
	if (val === null || val === undefined) {
		throw new TypeError('Object.assign cannot be called with null or undefined');
	}
	return Object(val);
}
function shouldUseNative() {
	try {
		if (!Object.assign) {
			return false;
		}
		var test1 = new String('abc');
		test1[5] = 'de';
		if (Object.getOwnPropertyNames(test1)[0] === '5') {
			return false;
		}
		var test2 = {};
		for (var i = 0; i < 10; i++) {
			test2['_' + String.fromCharCode(i)] = i;
		}
		var order2 = Object.getOwnPropertyNames(test2).map(function (n) {
			return test2[n];
		});
		if (order2.join('') !== '0123456789') {
			return false;
		}
		var test3 = {};
		'abcdefghijklmnopqrst'.split('').forEach(function (letter) {
			test3[letter] = letter;
		});
		if (Object.keys(Object.assign({}, test3)).join('') !==
				'abcdefghijklmnopqrst') {
			return false;
		}
		return true;
	} catch (err) {
		return false;
	}
}
var objectAssign = shouldUseNative() ? Object.assign : function (target, source) {
	var from;
	var to = toObject(target);
	var symbols;
	for (var s = 1; s < arguments.length; s++) {
		from = Object(arguments[s]);
		for (var key in from) {
			if (hasOwnProperty.call(from, key)) {
				to[key] = from[key];
			}
		}
		if (getOwnPropertySymbols) {
			symbols = getOwnPropertySymbols(from);
			for (var i = 0; i < symbols.length; i++) {
				if (propIsEnumerable.call(from, symbols[i])) {
					to[symbols[i]] = from[symbols[i]];
				}
			}
		}
	}
	return to;
};

var ReactPropTypesSecret = 'SECRET_DO_NOT_PASS_THIS_OR_YOU_WILL_BE_FIRED';
var ReactPropTypesSecret_1 = ReactPropTypesSecret;

var has = Function.call.bind(Object.prototype.hasOwnProperty);

function emptyFunction() {}
function emptyFunctionWithReset() {}
emptyFunctionWithReset.resetWarningCache = emptyFunction;
var factoryWithThrowingShims = function() {
  function shim(props, propName, componentName, location, propFullName, secret) {
    if (secret === ReactPropTypesSecret_1) {
      return;
    }
    var err = new Error(
      'Calling PropTypes validators directly is not supported by the `prop-types` package. ' +
      'Use PropTypes.checkPropTypes() to call them. ' +
      'Read more at http://fb.me/use-check-prop-types'
    );
    err.name = 'Invariant Violation';
    throw err;
  }  shim.isRequired = shim;
  function getShim() {
    return shim;
  }  var ReactPropTypes = {
    array: shim,
    bool: shim,
    func: shim,
    number: shim,
    object: shim,
    string: shim,
    symbol: shim,
    any: shim,
    arrayOf: getShim,
    element: shim,
    elementType: shim,
    instanceOf: getShim,
    node: shim,
    objectOf: getShim,
    oneOf: getShim,
    oneOfType: getShim,
    shape: getShim,
    exact: getShim,
    checkPropTypes: emptyFunctionWithReset,
    resetWarningCache: emptyFunction
  };
  ReactPropTypes.PropTypes = ReactPropTypes;
  return ReactPropTypes;
};

var propTypes = createCommonjsModule(function (module) {
{
  module.exports = factoryWithThrowingShims();
}
});

function _interopDefault (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes = _interopDefault(propTypes);
var _local_React = _interopDefault(React);
function _typeof(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }
function _objectSpread$2(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$2(target, key, source[key]); }); } return target; }
function _defineProperty$2(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$1(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$1(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$1(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle = {
  "willChange": "transform"
};
var AddAlt20 = _local_React.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$1(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$2({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$1(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof(style) === 'object') {
    props.style = _objectSpread$2({}, defaultStyle, style);
  } else {
    props.style = defaultStyle;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React.createElement('svg', props, children, _local_React.createElement('path', {
    d: 'M16 4A12 12 0 1 1 4 16 12 12 0 0 1 16 4m0-2a14 14 0 1 0 14 14A14 14 0 0 0 16 2z'
  }), _local_React.createElement('path', {
    d: 'M22 15h-5v-5h-2v5h-5v2h5v5h2v-5h5v-2z'
  }));
});
AddAlt20.displayName = 'AddAlt20';
AddAlt20.propTypes = {
  'aria-hidden': PropTypes.bool,
  'aria-label': PropTypes.string,
  'aria-labelledby': PropTypes.string,
  className: PropTypes.string,
  children: PropTypes.node,
  height: PropTypes.number,
  preserveAspectRatio: PropTypes.string,
  tabIndex: PropTypes.string,
  viewBox: PropTypes.string,
  width: PropTypes.number,
  xmlns: PropTypes.string
};
AddAlt20.defaultProps = {
  width: 20,
  height: 20,
  viewBox: '0 0 32 32',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _20 = AddAlt20;

function _interopDefault$1 (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes$1 = _interopDefault$1(propTypes);
var _local_React$1 = _interopDefault$1(React);
function _typeof$1(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof$1 = function _typeof(obj) { return typeof obj; }; } else { _typeof$1 = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof$1(obj); }
function _objectSpread$3(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$3(target, key, source[key]); }); } return target; }
function _defineProperty$3(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$2(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$2(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$2(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle$1 = {
  "willChange": "transform"
};
var SubtractAlt20 = _local_React$1.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$2(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$3({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$2(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof$1(style) === 'object') {
    props.style = _objectSpread$3({}, defaultStyle$1, style);
  } else {
    props.style = defaultStyle$1;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React$1.createElement('svg', props, children, _local_React$1.createElement('path', {
    d: 'M16 4A12 12 0 1 1 4 16 12 12 0 0 1 16 4m0-2a14 14 0 1 0 14 14A14 14 0 0 0 16 2z'
  }), _local_React$1.createElement('path', {
    d: 'M10 15h12v2H10z'
  }));
});
SubtractAlt20.displayName = 'SubtractAlt20';
SubtractAlt20.propTypes = {
  'aria-hidden': PropTypes$1.bool,
  'aria-label': PropTypes$1.string,
  'aria-labelledby': PropTypes$1.string,
  className: PropTypes$1.string,
  children: PropTypes$1.node,
  height: PropTypes$1.number,
  preserveAspectRatio: PropTypes$1.string,
  tabIndex: PropTypes$1.string,
  viewBox: PropTypes$1.string,
  width: PropTypes$1.number,
  xmlns: PropTypes$1.string
};
SubtractAlt20.defaultProps = {
  width: 20,
  height: 20,
  viewBox: '0 0 32 32',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _20$1 = SubtractAlt20;

function _interopDefault$2 (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes$2 = _interopDefault$2(propTypes);
var _local_React$2 = _interopDefault$2(React);
function _typeof$2(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof$2 = function _typeof(obj) { return typeof obj; }; } else { _typeof$2 = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof$2(obj); }
function _objectSpread$4(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$4(target, key, source[key]); }); } return target; }
function _defineProperty$4(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$3(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$3(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$3(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle$2 = {
  "willChange": "transform"
};
var ViewFilled20 = _local_React$2.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$3(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$4({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$3(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof$2(style) === 'object') {
    props.style = _objectSpread$4({}, defaultStyle$2, style);
  } else {
    props.style = defaultStyle$2;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React$2.createElement('svg', props, children, _local_React$2.createElement('circle', {
    cx: '16',
    cy: '16',
    r: '4'
  }), _local_React$2.createElement('path', {
    d: 'M30.94 15.66A16.69 16.69 0 0 0 16 5 16.69 16.69 0 0 0 1.06 15.66a1 1 0 0 0 0 .68A16.69 16.69 0 0 0 16 27a16.69 16.69 0 0 0 14.94-10.66 1 1 0 0 0 0-.68zM16 22.5a6.5 6.5 0 1 1 6.5-6.5 6.51 6.51 0 0 1-6.5 6.5z'
  }));
});
ViewFilled20.displayName = 'ViewFilled20';
ViewFilled20.propTypes = {
  'aria-hidden': PropTypes$2.bool,
  'aria-label': PropTypes$2.string,
  'aria-labelledby': PropTypes$2.string,
  className: PropTypes$2.string,
  children: PropTypes$2.node,
  height: PropTypes$2.number,
  preserveAspectRatio: PropTypes$2.string,
  tabIndex: PropTypes$2.string,
  viewBox: PropTypes$2.string,
  width: PropTypes$2.number,
  xmlns: PropTypes$2.string
};
ViewFilled20.defaultProps = {
  width: 20,
  height: 20,
  viewBox: '0 0 32 32',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _20$2 = ViewFilled20;

function _interopDefault$3 (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes$3 = _interopDefault$3(propTypes);
var _local_React$3 = _interopDefault$3(React);
function _typeof$3(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof$3 = function _typeof(obj) { return typeof obj; }; } else { _typeof$3 = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof$3(obj); }
function _objectSpread$5(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$5(target, key, source[key]); }); } return target; }
function _defineProperty$5(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$4(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$4(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$4(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle$3 = {
  "willChange": "transform"
};
var ViewOffFilled20 = _local_React$3.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$4(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$5({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$4(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof$3(style) === 'object') {
    props.style = _objectSpread$5({}, defaultStyle$3, style);
  } else {
    props.style = defaultStyle$3;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React$3.createElement('svg', props, children, _local_React$3.createElement('path', {
    d: 'M30.94 15.66a16.4 16.4 0 0 0-5.73-7.45L30 3.41 28.59 2 2 28.59 3.41 30l5.1-5.09A15.38 15.38 0 0 0 16 27a16.69 16.69 0 0 0 14.94-10.66 1 1 0 0 0 0-.68zM16 22.5a6.46 6.46 0 0 1-3.83-1.26L14 19.43A4 4 0 0 0 19.43 14l1.81-1.81A6.49 6.49 0 0 1 16 22.5zm-11.47-.69l5-5A6.84 6.84 0 0 1 9.5 16 6.51 6.51 0 0 1 16 9.5a6.84 6.84 0 0 1 .79.05l3.78-3.77A14.39 14.39 0 0 0 16 5 16.69 16.69 0 0 0 1.06 15.66a1 1 0 0 0 0 .68 15.86 15.86 0 0 0 3.47 5.47z'
  }));
});
ViewOffFilled20.displayName = 'ViewOffFilled20';
ViewOffFilled20.propTypes = {
  'aria-hidden': PropTypes$3.bool,
  'aria-label': PropTypes$3.string,
  'aria-labelledby': PropTypes$3.string,
  className: PropTypes$3.string,
  children: PropTypes$3.node,
  height: PropTypes$3.number,
  preserveAspectRatio: PropTypes$3.string,
  tabIndex: PropTypes$3.string,
  viewBox: PropTypes$3.string,
  width: PropTypes$3.number,
  xmlns: PropTypes$3.string
};
ViewOffFilled20.defaultProps = {
  width: 20,
  height: 20,
  viewBox: '0 0 32 32',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _20$3 = ViewOffFilled20;

function styleInject(css, ref) {
  if ( ref === void 0 ) ref = {};
  var insertAt = ref.insertAt;
  if (!css || typeof document === 'undefined') { return; }
  var head = document.head || document.getElementsByTagName('head')[0];
  var style = document.createElement('style');
  style.type = 'text/css';
  if (insertAt === 'top') {
    if (head.firstChild) {
      head.insertBefore(style, head.firstChild);
    } else {
      head.appendChild(style);
    }
  } else {
    head.appendChild(style);
  }
  if (style.styleSheet) {
    style.styleSheet.cssText = css;
  } else {
    style.appendChild(document.createTextNode(css));
  }
}

var css = ".webhook-create .bx--inline-notification {\n  width: 100%;\n  max-width: 100%;\n  min-width: 100%; }\n\n.create-container {\n  padding-top: 3rem;\n  background-color: white; }\n  .create-container .title {\n    font-size: 150%;\n    text-align: center;\n    padding-bottom: 3rem; }\n  .create-container .row {\n    display: flex;\n    flex-wrap: nowrap;\n    align-items: center;\n    padding: 1% 0% 1% 15%; }\n  .create-container .item-label {\n    width: 15%;\n    background-color: white;\n    text-align: left; }\n  .create-container .help-icon {\n    padding-right: 1%;\n    width: 4%; }\n  .create-container .entry-field {\n    background-color: white;\n    width: 60%; }\n  .create-container .git-access-drop-down-div {\n    background-color: white;\n    width: 50%; }\n  .create-container .secButtonEnabled {\n    cursor: pointer; }\n  .create-container .secButtonDisabled {\n    cursor: unset;\n    fill: #dcdcdc;\n    pointer-events: none; }\n  .create-container #pipeline, .create-container #namespace, .create-container #git, .create-container #serviceAccounts {\n    margin: 0; }\n  .create-container .add-sec-btn, .create-container .del-sec-btn {\n    width: 5%;\n    display: flex;\n    align-items: center;\n    justify-content: center; }\n  .create-container .add-sec-btn {\n    fill: green; }\n  .create-container .del-sec-btn {\n    fill: red; }\n  .create-container .token {\n    align-items: center;\n    display: flex; }\n  .create-container .token-visible, .create-container .token-invisible {\n    right: 2.25rem;\n    position: fixed;\n    cursor: pointer; }\n  .create-container .token-visible {\n    visibility: visible; }\n  .create-container .token-invisible {\n    visibility: hidden; }\n  .create-container #cancel, .create-container #submit {\n    font-weight: bold;\n    float: left;\n    padding-left: 15%;\n    padding-right: 15%; }\n  .create-container #cancel {\n    background-color: red; }\n  .create-container #submit {\n    background-color: green; }\n  .create-container #del #add {\n    padding: 0; }\n  .create-container .bx--text-input__field-wrapper {\n    width: 100%; }\n  .create-container .bx--modal-content {\n    width: 100%;\n    padding-left: 1rem;\n    padding-right: 1rem; }\n  .create-container .bx--modal.is-visible {\n    background-color: rgba(255, 255, 255, 0.82); }\n  .create-container .bx--modal-container {\n    border: 1px solid black;\n    background-color: white; }\n  .create-container .bx--btn.bx--btn--primary, .create-container .bx--btn.bx--btn--secondary {\n    display: inline;\n    text-align: center;\n    padding: 0px;\n    font-weight: bold; }\n  .create-container .bx--btn.bx--btn--primary {\n    background-color: green; }\n  .create-container .bx--btn.bx--btn--secondary {\n    background-color: red; }\n  .create-container .bx--btn.bx--btn--primary.bx--btn--disabled {\n    background-color: lightgrey;\n    font-weight: bold;\n    float: right;\n    padding-left: 15%;\n    padding-right: 15%;\n    color: white; }\n  .create-container .modal-row {\n    display: flex;\n    align-items: center;\n    padding: 1%; }\n  .create-container .modal-row-help-icon {\n    width: 8%; }\n  .create-container .modal-row-item-label {\n    width: 30%; }\n  .create-container .modal-row-entry-field {\n    width: 60%; }\n";
styleInject(css);

var WebhookCreatePage =
function (_Component) {
  _inherits(WebhookCreatePage, _Component);
  function WebhookCreatePage(props) {
    var _this;
    _classCallCheck(this, WebhookCreatePage);
    _this = _possibleConstructorReturn(this, _getPrototypeOf(WebhookCreatePage).call(this, props));
    _defineProperty(_assertThisInitialized(_this), "handleChange", function (event) {
      var target = event.target;
      var value = target.value;
      var name = target.name;
      _this.setState(_defineProperty({}, name, value));
    });
    _defineProperty(_assertThisInitialized(_this), "handleChangeNamespace", function (itemText, value) {
      _this.setState({
        namespace: itemText.selectedItem,
        apiPipelines: '',
        apiSecrets: '',
        apiServiceAccounts: '',
        pipeline: '',
        gitsecret: '',
        serviceAccount: ''
      });
      if (!_this.state.pipelineFail) {
        _this.fetchPipelines(itemText.selectedItem);
      }
      if (!_this.state.secretsFail) {
        _this.fetchSecrets(itemText.selectedItem);
      }
      if (!_this.state.serviceAccountsFail) {
        _this.fetchServiceAccounts(itemText.selectedItem);
      }
    });
    _defineProperty(_assertThisInitialized(_this), "handleChangePipeline", function (itemText, value) {
      _this.setState({
        pipeline: itemText.selectedItem
      });
    });
    _defineProperty(_assertThisInitialized(_this), "handleChangeSecret", function (itemText, value) {
      _this.setState({
        gitsecret: itemText.selectedItem
      });
    });
    _defineProperty(_assertThisInitialized(_this), "handleChangeServiceAcct", function (itemText, value) {
      _this.setState({
        serviceAccount: itemText.selectedItem
      });
    });
    _defineProperty(_assertThisInitialized(_this), "handleSubmit", function (e) {
      e.preventDefault();
      var requestBody = {
        name: _this.state.name,
        gitrepositoryurl: _this.state.repository,
        accesstoken: _this.state.gitsecret,
        pipeline: _this.state.pipeline,
        namespace: _this.state.namespace,
        serviceaccount: _this.state.serviceAccount,
        dockerregistry: _this.state.dockerRegistry
      };
      createWebhook(requestBody).then(function () {
        _this.props.setShowNotificationOnTable(true);
        _this.returnToTable();
      })["catch"](function (error) {
        error.response.text().then(function (text) {
          _this.setState({
            notificationErrorMessage: "Failed to create webhook, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true
          });
        });
      });
    });
    _defineProperty(_assertThisInitialized(_this), "returnToTable", function () {
      var cutpoint = _this.props.match.url.lastIndexOf('/');
      var matchURL = _this.props.match.url.slice(0, cutpoint);
      _this.props.history.push(matchURL);
    });
    _defineProperty(_assertThisInitialized(_this), "isDisabled", function () {
      if (_this.state.namespace === "") {
        return true;
      }
      return false;
    });
    _defineProperty(_assertThisInitialized(_this), "isFormIncomplete", function () {
      if (!_this.state.name || !_this.state.repository || !_this.state.namespace || !_this.state.pipeline || !_this.state.gitsecret || !_this.state.serviceAccount || !_this.state.dockerRegistry) {
        return true;
      }
      return false;
    });
    _defineProperty(_assertThisInitialized(_this), "createButtonIDForCSS", function () {
      if (_this.isFormIncomplete()) {
        return "disable";
      }
      return "submit";
    });
    _defineProperty(_assertThisInitialized(_this), "displayNamespaceDropDown", function (namespaceItems) {
      if (!_this.state.apiNamespaces) {
        return React.createElement(CarbonComponentsReact.DropdownSkeleton, null);
      }
      return React.createElement(CarbonComponentsReact.Dropdown, {
        id: "namespace",
        label: "select namespace",
        items: namespaceItems,
        tabIndex: 5,
        onChange: _this.handleChangeNamespace
      });
    });
    _defineProperty(_assertThisInitialized(_this), "displayPipelineDropDown", function (pipelineItems) {
      if (!_this.isDisabled()) {
        if (!_this.state.apiPipelines) {
          return React.createElement(CarbonComponentsReact.DropdownSkeleton, null);
        }
      }
      return React.createElement(CarbonComponentsReact.Dropdown, {
        id: "pipeline",
        label: "select pipeline",
        items: pipelineItems,
        tabIndex: 7,
        disabled: _this.isDisabled(),
        onChange: _this.handleChangePipeline
      });
    });
    _defineProperty(_assertThisInitialized(_this), "displaySecretDropDown", function (secretItems) {
      if (!_this.isDisabled()) {
        if (!_this.state.apiSecrets) {
          return React.createElement(CarbonComponentsReact.DropdownSkeleton, null);
        }
      }
      return React.createElement(CarbonComponentsReact.Dropdown, {
        id: "git",
        label: "select secret",
        items: secretItems,
        tabIndex: 9,
        disabled: _this.isDisabled(),
        onChange: _this.handleChangeSecret,
        selectedItem: _this.state.gitsecret
      });
    });
    _defineProperty(_assertThisInitialized(_this), "displayServiceAccountDropDown", function (saItems) {
      if (!_this.isDisabled()) {
        if (!_this.state.apiServiceAccounts) {
          return React.createElement(CarbonComponentsReact.DropdownSkeleton, null);
        }
      }
      return React.createElement(CarbonComponentsReact.Dropdown, {
        id: "serviceAccounts",
        label: "select service account",
        items: saItems,
        tabIndex: 11,
        disabled: _this.isDisabled(),
        onChange: _this.handleChangeServiceAcct
      });
    });
    _defineProperty(_assertThisInitialized(_this), "getSecretButtonCSSID", function () {
      if (_this.isDisabled()) {
        return "secButtonDisabled";
      }
      return "secButtonEnabled";
    });
    _defineProperty(_assertThisInitialized(_this), "toggleDeleteDialog", function () {
      if (_this.state.gitsecret) {
        var invert = !_this.state.showDeleteDialog;
        _this.setState({
          showDeleteDialog: invert,
          showNotification: false
        });
      } else {
        _this.setState({
          showNotification: true,
          notificationErrorMessage: "No secret selected. A secret must be selected from the drop down before selecting delete.",
          notificationStatus: "error",
          notificationStatusMsgShort: "Error:"
        });
      }
    });
    _defineProperty(_assertThisInitialized(_this), "toggleCreateDialog", function () {
      if (_this.state.showNotification) {
        _this.setState({
          showNotification: false
        });
      }
      var invert = !_this.state.showCreateDialog;
      _this.setState({
        showCreateDialog: invert,
        newSecretName: '',
        newTokenValue: ''
      });
    });
    _defineProperty(_assertThisInitialized(_this), "deleteAccessTokenSecret", function () {
      deleteSecret(_this.state.gitsecret, _this.state.namespace).then(function () {
        _this.toggleDeleteDialog();
        _this.setState({
          apiSecrets: '',
          gitsecret: '',
          showNotification: true,
          notificationErrorMessage: "",
          notificationStatus: "success",
          notificationStatusMsgShort: "Secret deleted."
        });
      })["catch"](function (error) {
        error.response.text().then(function (text) {
          _this.toggleDeleteDialog();
          _this.setState({
            notificationErrorMessage: "Failed to delete secret, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true
          });
        });
      })["finally"](function () {
        _this.fetchSecrets(_this.state.namespace);
      });
    });
    _defineProperty(_assertThisInitialized(_this), "createAccessTokenSecret", function () {
      var requestBody = {
        name: _this.state.newSecretName,
        accesstoken: _this.state.newTokenValue
      };
      createSecret(requestBody, _this.state.namespace).then(function () {
        _this.toggleCreateDialog();
        _this.setState({
          gitsecret: _this.state.newSecretName,
          newSecretName: '',
          newTokenValue: '',
          showNotification: true,
          notificationErrorMessage: "",
          notificationStatus: "success",
          notificationStatusMsgShort: "Secret created."
        });
      })["catch"](function (error) {
        error.response.text().then(function (text) {
          _this.toggleCreateDialog();
          _this.setState({
            newSecretName: '',
            newTokenValue: '',
            notificationErrorMessage: "Failed to create secret, error returned was : " + text,
            notificationStatus: 'error',
            notificationStatusMsgShort: 'Error:',
            showNotification: true
          });
        });
      })["finally"](function () {
        _this.fetchSecrets(_this.state.namespace);
      });
    });
    _defineProperty(_assertThisInitialized(_this), "handleModalText", function (event) {
      if (event) {
        var target = event.target;
        var value = target.value;
        var name = target.name;
        _this.setState(_defineProperty({}, name, value));
      }
    });
    _defineProperty(_assertThisInitialized(_this), "togglePasswordVisibility", function () {
      _this.setState({
        pwType: _this.state.pwType === 'password' ? 'text' : 'password',
        visibleCSS: _this.state.visibleCSS === 'token-visible' ? 'token-invisible' : 'token-visible',
        invisibleCSS: _this.state.invisibleCSS === 'token-invisible' ? 'token-visible' : 'token-invisible'
      });
    });
    _this.props.setShowNotificationOnTable(false);
    _this.state = {
      namespaceFail: false,
      pipelineFail: false,
      secretsFail: false,
      serviceAccountsFail: false,
      name: '',
      repository: '',
      namespace: '',
      pipeline: '',
      gitsecret: '',
      serviceAccount: '',
      dockerRegistry: '',
      apiNamespaces: '',
      apiPipelines: '',
      apiSecrets: '',
      apiServiceAccounts: '',
      showDeleteDialog: false,
      showCreateDialog: false,
      showNotification: false,
      notificationErrorMessage: "",
      notificationStatus: 'success',
      notificationStatusMsgShort: 'Secret deleted successfully',
      newSecretName: '',
      newTokenValue: '',
      createSecretDisabled: true,
      pwType: 'password',
      visibleCSS: 'token-visible',
      invisibleCSS: 'token-invisible'
    };
    return _this;
  }
  _createClass(WebhookCreatePage, [{
    key: "fetchNamespaces",
    value: function () {
      var _fetchNamespaces = _asyncToGenerator(
      regeneratorRuntime.mark(function _callee() {
        var _this2 = this;
        var ns;
        return regeneratorRuntime.wrap(function _callee$(_context) {
          while (1) {
            switch (_context.prev = _context.next) {
              case 0:
                _context.prev = 0;
                _context.next = 3;
                return getNamespaces();
              case 3:
                ns = _context.sent;
                this.setState({
                  apiNamespaces: ns
                });
                _context.next = 10;
                break;
              case 7:
                _context.prev = 7;
                _context.t0 = _context["catch"](0);
                _context.t0.response.text().then(function (text) {
                  _this2.setState({
                    namespaceFail: true,
                    notificationErrorMessage: "Failed to get namespaces, error returned was : " + text,
                    notificationStatus: 'error',
                    notificationStatusMsgShort: 'Error:',
                    showNotification: true
                  });
                });
              case 10:
              case "end":
                return _context.stop();
            }
          }
        }, _callee, this, [[0, 7]]);
      }));
      function fetchNamespaces() {
        return _fetchNamespaces.apply(this, arguments);
      }
      return fetchNamespaces;
    }()
  }, {
    key: "fetchPipelines",
    value: function () {
      var _fetchPipelines = _asyncToGenerator(
      regeneratorRuntime.mark(function _callee2(namespace) {
        var _this3 = this;
        var pl;
        return regeneratorRuntime.wrap(function _callee2$(_context2) {
          while (1) {
            switch (_context2.prev = _context2.next) {
              case 0:
                _context2.prev = 0;
                _context2.next = 3;
                return getPipelines(namespace);
              case 3:
                pl = _context2.sent;
                this.setState({
                  apiPipelines: pl
                });
                _context2.next = 10;
                break;
              case 7:
                _context2.prev = 7;
                _context2.t0 = _context2["catch"](0);
                _context2.t0.response.text().then(function (text) {
                  _this3.setState({
                    pipelineFail: true,
                    notificationErrorMessage: "Failed to get pipelines, error returned was : " + text,
                    notificationStatus: 'error',
                    notificationStatusMsgShort: 'Error:',
                    showNotification: true
                  });
                });
              case 10:
              case "end":
                return _context2.stop();
            }
          }
        }, _callee2, this, [[0, 7]]);
      }));
      function fetchPipelines(_x) {
        return _fetchPipelines.apply(this, arguments);
      }
      return fetchPipelines;
    }()
  }, {
    key: "fetchSecrets",
    value: function () {
      var _fetchSecrets = _asyncToGenerator(
      regeneratorRuntime.mark(function _callee3(namespace) {
        var _this4 = this;
        var s;
        return regeneratorRuntime.wrap(function _callee3$(_context3) {
          while (1) {
            switch (_context3.prev = _context3.next) {
              case 0:
                _context3.prev = 0;
                _context3.next = 3;
                return getSecrets(namespace);
              case 3:
                s = _context3.sent;
                this.setState({
                  apiSecrets: s
                });
                _context3.next = 10;
                break;
              case 7:
                _context3.prev = 7;
                _context3.t0 = _context3["catch"](0);
                _context3.t0.response.text().then(function (text) {
                  _this4.setState({
                    secretsFail: true,
                    notificationErrorMessage: "Failed to get secrets, error returned was : " + text,
                    notificationStatus: 'error',
                    notificationStatusMsgShort: 'Error:',
                    showNotification: true
                  });
                });
              case 10:
              case "end":
                return _context3.stop();
            }
          }
        }, _callee3, this, [[0, 7]]);
      }));
      function fetchSecrets(_x2) {
        return _fetchSecrets.apply(this, arguments);
      }
      return fetchSecrets;
    }()
  }, {
    key: "fetchServiceAccounts",
    value: function () {
      var _fetchServiceAccounts = _asyncToGenerator(
      regeneratorRuntime.mark(function _callee4(namespace) {
        var _this5 = this;
        var sa;
        return regeneratorRuntime.wrap(function _callee4$(_context4) {
          while (1) {
            switch (_context4.prev = _context4.next) {
              case 0:
                _context4.prev = 0;
                _context4.next = 3;
                return getServiceAccounts(namespace);
              case 3:
                sa = _context4.sent;
                this.setState({
                  apiServiceAccounts: sa
                });
                _context4.next = 10;
                break;
              case 7:
                _context4.prev = 7;
                _context4.t0 = _context4["catch"](0);
                _context4.t0.response.text().then(function (text) {
                  _this5.setState({
                    serviceAccountsFail: true,
                    notificationErrorMessage: "Failed to get service accounts, error returned was : " + text,
                    notificationStatus: 'error',
                    notificationStatusMsgShort: 'Error:',
                    showNotification: true
                  });
                });
              case 10:
              case "end":
                return _context4.stop();
            }
          }
        }, _callee4, this, [[0, 7]]);
      }));
      function fetchServiceAccounts(_x3) {
        return _fetchServiceAccounts.apply(this, arguments);
      }
      return fetchServiceAccounts;
    }()
  }, {
    key: "render",
    value: function render() {
      var _this6 = this;
      var namespaceItems = [];
      var pipelineItems = [];
      var secretItems = [];
      var saItems = [];
      if (!this.state.apiNamespaces) {
        if (!this.state.namespaceFail) {
          this.fetchNamespaces();
        }
      } else {
        this.state.apiNamespaces.items.map(function (namespaceResource, index) {
          namespaceItems[index] = namespaceResource.metadata['name'];
        });
        if (this.state.apiPipelines) {
          this.state.apiPipelines.items.map(function (pipelineResource, index) {
            pipelineItems[index] = pipelineResource.metadata['name'];
          });
        }
        if (this.state.apiSecrets) {
          this.state.apiSecrets.map(function (secretResource, index) {
            secretItems[index] = secretResource['name'];
          });
        }
        if (this.state.apiServiceAccounts) {
          this.state.apiServiceAccounts.items.map(function (saResource, index) {
            saItems[index] = saResource.metadata['name'];
          });
        }
        if (this.state.createSecretDisabled) {
          if (this.state.newSecretName && this.state.newTokenValue) {
            this.setState({
              createSecretDisabled: false
            });
          }
        } else {
          if (!this.state.newSecretName || !this.state.newTokenValue) {
            this.setState({
              createSecretDisabled: true
            });
          }
        }
      }
      return React.createElement("div", {
        className: "webhook-create"
      }, React.createElement("div", {
        className: "notification"
      }, this.state.showNotification && React.createElement(CarbonComponentsReact.InlineNotification, {
        kind: this.state.notificationStatus,
        subtitle: this.state.notificationErrorMessage,
        title: this.state.notificationStatusMsgShort
      }), this.state.showNotification && window.scrollTo(0, 0)), React.createElement("div", {
        className: "create-container"
      }, React.createElement(CarbonComponentsReact.Form, {
        onSubmit: this.handleSubmit
      }, React.createElement("div", {
        className: "title"
      }, "Create Webhook"), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "name-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 0
      }, React.createElement("p", null, "The display name for your webhook in this user interface."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Name")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createTextEntry"
      }, React.createElement(CarbonComponentsReact.TextInput, {
        id: "id",
        placeholder: "Enter display name here",
        name: "name",
        value: this.state.name,
        onChange: this.handleChange,
        tabIndex: 1,
        hideLabel: true,
        labelText: "Display Name",
        "data-testid": "display-name-entry"
      })))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "git-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 2
      }, React.createElement("p", null, "The URL of the git repository to create the webhook on."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Repository URL")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createTextEntry"
      }, React.createElement(CarbonComponentsReact.TextInput, {
        id: "git-repo",
        placeholder: "https://github.com/org/repo.git",
        name: "repository",
        value: this.state.repo,
        onChange: this.handleChange,
        tabIndex: 3,
        hideLabel: true,
        labelText: "Repository",
        "data-testid": "git-url-entry"
      })))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "namespace-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 4
      }, React.createElement("p", null, "The namespace to operate in."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Namespace")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createDropDown"
      }, this.displayNamespaceDropDown(namespaceItems)))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "pipeline-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 6
      }, React.createElement("p", null, "The pipeline from the selected namespace to run when the webhook is triggered."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Pipeline")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createDropDown"
      }, this.displayPipelineDropDown(pipelineItems)))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "secret-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 8
      }, React.createElement("p", null, "The kubernetes secret holding access information for the git repository. The credential must have sufficient privileges to create webhooks in the repository."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Access Token")), React.createElement("div", {
        className: "del-sec-btn"
      }, React.createElement(_20$1, {
        id: "delete-secret-button",
        className: this.getSecretButtonCSSID(),
        onClick: function onClick() {
          _this6.toggleDeleteDialog();
        }
      })), React.createElement("div", {
        className: "git-access-drop-down-div"
      }, React.createElement("div", {
        className: "createDropDown"
      }, this.displaySecretDropDown(secretItems))), React.createElement("div", {
        className: "add-sec-btn"
      }, React.createElement(_20, {
        id: "create-secret-button",
        className: this.getSecretButtonCSSID(),
        onClick: function onClick() {
          _this6.toggleCreateDialog();
        }
      }))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "serviceaccount-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 10
      }, React.createElement("p", null, "The service account under which to run the pipeline run."), React.createElement("br", null), React.createElement("p", null, "The service account needs to be patched with secrets to access both git and docker."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Service Account")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createDropDown"
      }, this.displayServiceAccountDropDown(saItems)))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon",
        id: "docker-tooltip"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 12
      }, React.createElement("p", null, "The docker registry to push images to."))), React.createElement("div", {
        className: "item-label"
      }, React.createElement("div", {
        className: "createLabel"
      }, "Docker Registry")), React.createElement("div", {
        className: "entry-field"
      }, React.createElement("div", {
        className: "createTextEntry"
      }, React.createElement(CarbonComponentsReact.TextInput, {
        id: "registry",
        placeholder: "Enter docker registry here",
        name: "dockerRegistry",
        value: this.state.dockerRegistry,
        onChange: this.handleChange,
        hideLabel: true,
        labelText: "Docker Registry",
        "data-testid": "docker-reg-entry"
      })))), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon"
      }), React.createElement("div", {
        className: "item-label"
      }), React.createElement("div", {
        className: "entry-field"
      })), React.createElement("div", {
        className: "row"
      }, React.createElement("div", {
        className: "help-icon"
      }), React.createElement("div", {
        className: "item-label"
      }), React.createElement("div", {
        className: "entry-field"
      }, React.createElement(CarbonComponentsReact.Button, {
        "data-testid": "cancel-button",
        id: "cancel",
        tabIndex: 13,
        onClick: function onClick() {
          _this6.returnToTable();
        }
      }, "Cancel"), React.createElement(CarbonComponentsReact.Button, {
        "data-testid": "create-button",
        type: "submit",
        tabIndex: 14,
        id: this.createButtonIDForCSS(),
        disabled: this.isFormIncomplete()
      }, "Create")))), React.createElement("div", {
        className: "delete-modal"
      }, React.createElement(CarbonComponentsReact.Modal, {
        open: this.state.showDeleteDialog,
        id: "delete-modal",
        modalLabel: "",
        modalHeading: "Please confirm you want to delete the following secret:",
        primaryButtonText: "Confirm",
        secondaryButtonText: "Cancel",
        danger: false,
        onSecondarySubmit: function onSecondarySubmit() {
          return _this6.toggleDeleteDialog();
        },
        onRequestSubmit: function onRequestSubmit() {
          return _this6.deleteAccessTokenSecret();
        },
        onRequestClose: function onRequestClose() {
          return _this6.toggleDeleteDialog();
        }
      }, React.createElement("div", {
        className: "secret-to-delete"
      }, this.state.gitsecret))), React.createElement("div", {
        className: "create-modal"
      }, React.createElement(CarbonComponentsReact.Modal, {
        open: this.state.showCreateDialog,
        id: "create-modal",
        modalLabel: "",
        modalHeading: "",
        primaryButtonText: "Create",
        primaryButtonDisabled: this.state.createSecretDisabled,
        secondaryButtonText: "Cancel",
        danger: false,
        onSecondarySubmit: function onSecondarySubmit() {
          return _this6.toggleCreateDialog();
        },
        onRequestSubmit: function onRequestSubmit() {
          return _this6.createAccessTokenSecret();
        },
        onRequestClose: function onRequestClose() {
          return _this6.toggleCreateDialog();
        }
      }, React.createElement("div", {
        className: "title"
      }, "Create Access Token Secret"), React.createElement("div", {
        className: "modal-row"
      }, React.createElement("div", {
        className: "modal-row-help-icon"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 15
      }, React.createElement("p", null, "The name of the secret to create."))), React.createElement("div", {
        className: "modal-row-item-label"
      }, React.createElement("div", null, "Name")), React.createElement("div", {
        className: "modal-row-entry-field"
      }, React.createElement("div", {
        className: ""
      }, React.createElement(CarbonComponentsReact.TextInput, {
        id: "secretName",
        placeholder: "Enter secret name here",
        name: "newSecretName",
        type: "text",
        value: this.state.newSecretName,
        onChange: this.handleModalText,
        hideLabel: true,
        labelText: "Secret Name",
        tabIndex: 16
      })))), React.createElement("div", {
        className: "modal-row"
      }, React.createElement("div", {
        className: "modal-row-help-icon"
      }, React.createElement(CarbonComponentsReact.Tooltip, {
        direction: "bottom",
        triggerText: "",
        tabIndex: 17
      }, React.createElement("p", null, "The access token."))), React.createElement("div", {
        className: "modal-row-item-label"
      }, React.createElement("div", null, "Access Token")), React.createElement("div", {
        className: "modal-row-entry-field"
      }, React.createElement("div", {
        className: "token"
      }, React.createElement(CarbonComponentsReact.TextInput, {
        id: "tokenValue",
        placeholder: "Enter access token here",
        name: "newTokenValue",
        type: this.state.pwType,
        value: this.state.newTokenValue,
        onChange: this.handleModalText,
        hideLabel: true,
        labelText: "Access Token",
        tabIndex: 18
      }), React.createElement(_20$2, {
        id: "token-visible-svg",
        className: this.state.visibleCSS,
        onClick: this.togglePasswordVisibility
      }), React.createElement(_20$3, {
        id: "token-invisible-svg",
        className: this.state.invisibleCSS,
        onClick: this.togglePasswordVisibility
      }))))))));
    }
  }]);
  return WebhookCreatePage;
}(React.Component);
var WebhookCreate = ReactRouterDOM.withRouter(WebhookCreatePage);

var css$1 = ".spinner-div {\n  padding-left: 40%;\n  padding-top: 15%; }\n\n.table-container #toolbar {\n  background: transparent; }\n\n.table-container #create-btn {\n  background-color: green;\n  font-weight: bold;\n  text-align: center;\n  margin: auto;\n  width: 100%;\n  padding: 0% 10% 0% 10%;\n  height: 100%;\n  display: inline-flex; }\n\n.table-container #delete-btn {\n  padding: 0.875rem 2.5rem 0.875rem 0.5rem; }\n\n.table-container .create-icon {\n  fill: white;\n  padding-top: 5%; }\n\n.table-container .search-bar {\n  width: 100%;\n  display: contents; }\n\n.table-container .btn-div {\n  width: 10%; }\n\n.table-container .bx--btn--ghost {\n  background-color: white; }\n\n.table-container .bx--batch-actions--active {\n  justify-content: flex-end; }\n\n.table-container .bx--data-table-header {\n  background: #f4f7fb; }\n\n.table-container .bx--table-toolbar {\n  border-left: 2px solid #f4f7fb;\n  border-right: 3px solid #f4f7fb;\n  background: #f4f7fb; }\n\n.table-container .bx--toolbar-search-container-expandable .bx--search {\n  width: 100%; }\n\n.table-container .bx--data-table th {\n  background-color: #7190c1;\n  border: 2px solid white; }\n\n.table-container .bx--table-sort {\n  background-color: #7190c1;\n  padding: 0px; }\n\n.table-container .bx--table-sort:focus {\n  outline: transparent; }\n\n.table-container .bx--btn--primary {\n  color: #ffffff;\n  white-space: nowrap; }\n\n.table-container .bx--data-table--zebra tbody tr:nth-child(odd) td {\n  border: 0, 2px, 2px, 2px solid white; }\n\n.table-container .bx--data-table--zebra tbody tr:nth-child(even) td {\n  border: 2px solid white; }\n\n.table-container .bx--inline-notification {\n  max-width: 100%; }\n";
styleInject(css$1);

function _interopDefault$4 (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes$4 = _interopDefault$4(propTypes);
var _local_React$4 = _interopDefault$4(React);
function _typeof$4(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof$4 = function _typeof(obj) { return typeof obj; }; } else { _typeof$4 = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof$4(obj); }
function _objectSpread$6(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$6(target, key, source[key]); }); } return target; }
function _defineProperty$6(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$5(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$5(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$5(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle$4 = {
  "willChange": "transform"
};
var Delete16 = _local_React$4.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$5(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$6({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$5(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof$4(style) === 'object') {
    props.style = _objectSpread$6({}, defaultStyle$4, style);
  } else {
    props.style = defaultStyle$4;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React$4.createElement('svg', props, children, _local_React$4.createElement('path', {
    d: 'M6 6h1v6H6zm3 0h1v6H9z'
  }), _local_React$4.createElement('path', {
    d: 'M2 3v1h1v10c0 .6.4 1 1 1h8c.6 0 1-.4 1-1V4h1V3H2zm2 11V4h8v10H4zM6 1h4v1H6z'
  }));
});
Delete16.displayName = 'Delete16';
Delete16.propTypes = {
  'aria-hidden': PropTypes$4.bool,
  'aria-label': PropTypes$4.string,
  'aria-labelledby': PropTypes$4.string,
  className: PropTypes$4.string,
  children: PropTypes$4.node,
  height: PropTypes$4.number,
  preserveAspectRatio: PropTypes$4.string,
  tabIndex: PropTypes$4.string,
  viewBox: PropTypes$4.string,
  width: PropTypes$4.number,
  xmlns: PropTypes$4.string
};
Delete16.defaultProps = {
  width: 16,
  height: 16,
  viewBox: '0 0 16 16',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _16 = Delete16;

function _interopDefault$5 (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var PropTypes$5 = _interopDefault$5(propTypes);
var _local_React$5 = _interopDefault$5(React);
function _typeof$5(obj) { if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof$5 = function _typeof(obj) { return typeof obj; }; } else { _typeof$5 = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof$5(obj); }
function _objectSpread$7(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty$7(target, key, source[key]); }); } return target; }
function _defineProperty$7(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _objectWithoutProperties$6(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose$6(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose$6(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
var defaultStyle$5 = {
  "willChange": "transform"
};
var AddAlt16 = _local_React$5.forwardRef(function (_ref, ref) {
  var className = _ref.className,
      children = _ref.children,
      style = _ref.style,
      tabIndex = _ref.tabIndex,
      rest = _objectWithoutProperties$6(_ref, ["className", "children", "style", "tabIndex"]);
  var _getAttributes = es.getAttributes(_objectSpread$7({}, rest, {
    tabindex: tabIndex
  })),
      tabindex = _getAttributes.tabindex,
      props = _objectWithoutProperties$6(_getAttributes, ["tabindex"]);
  if (className) {
    props.className = className;
  }
  if (tabindex !== undefined && tabindex !== null) {
    props.tabIndex = tabindex;
  }
  if (_typeof$5(style) === 'object') {
    props.style = _objectSpread$7({}, defaultStyle$5, style);
  } else {
    props.style = defaultStyle$5;
  }
  if (ref) {
    props.ref = ref;
  }
  return _local_React$5.createElement('svg', props, children, _local_React$5.createElement('path', {
    d: 'M8 2c3.3 0 6 2.7 6 6s-2.7 6-6 6-6-2.7-6-6 2.7-6 6-6m0-1C4.1 1 1 4.1 1 8s3.1 7 7 7 7-3.1 7-7-3.1-7-7-7z'
  }), _local_React$5.createElement('path', {
    d: 'M11 7.5H8.5V5h-1v2.5H5v1h2.5V11h1V8.5H11z'
  }));
});
AddAlt16.displayName = 'AddAlt16';
AddAlt16.propTypes = {
  'aria-hidden': PropTypes$5.bool,
  'aria-label': PropTypes$5.string,
  'aria-labelledby': PropTypes$5.string,
  className: PropTypes$5.string,
  children: PropTypes$5.node,
  height: PropTypes$5.number,
  preserveAspectRatio: PropTypes$5.string,
  tabIndex: PropTypes$5.string,
  viewBox: PropTypes$5.string,
  width: PropTypes$5.number,
  xmlns: PropTypes$5.string
};
AddAlt16.defaultProps = {
  width: 16,
  height: 16,
  viewBox: '0 0 16 16',
  xmlns: 'http://www.w3.org/2000/svg',
  preserveAspectRatio: 'xMidYMid meet'
};
var _16$1 = AddAlt16;

var TableContainer = CarbonComponentsReact.DataTable.TableContainer,
    Table = CarbonComponentsReact.DataTable.Table,
    TableHead = CarbonComponentsReact.DataTable.TableHead,
    TableRow = CarbonComponentsReact.DataTable.TableRow,
    TableBody = CarbonComponentsReact.DataTable.TableBody,
    TableCell = CarbonComponentsReact.DataTable.TableCell,
    TableHeader = CarbonComponentsReact.DataTable.TableHeader;
var WebhookDisplayTable =
function (_Component) {
  _inherits(WebhookDisplayTable, _Component);
  function WebhookDisplayTable(props) {
    var _this;
    _classCallCheck(this, WebhookDisplayTable);
    _this = _possibleConstructorReturn(this, _getPrototypeOf(WebhookDisplayTable).call(this, props));
    _defineProperty(_assertThisInitialized(_this), "handleSelectedRows", function (rows) {
      console.log(rows);
    });
    _this.state = {
      error: null,
      isLoaded: false,
      webhooks: [],
      showNotification: false,
      notificationErrorMessage: '',
      notificationStatus: '',
      notificationStatusMsgShort: ''
    };
    return _this;
  }
  _createClass(WebhookDisplayTable, [{
    key: "componentDidMount",
    value: function () {
      var _componentDidMount = _asyncToGenerator(
      regeneratorRuntime.mark(function _callee() {
        var _this2 = this;
        var data;
        return regeneratorRuntime.wrap(function _callee$(_context) {
          while (1) {
            switch (_context.prev = _context.next) {
              case 0:
                _context.prev = 0;
                _context.next = 3;
                return getWebhooks();
              case 3:
                data = _context.sent;
                this.setState({
                  isLoaded: true,
                  webhooks: data
                });
                _context.next = 10;
                break;
              case 7:
                _context.prev = 7;
                _context.t0 = _context["catch"](0);
                _context.t0.response.text().then(function (text) {
                  _this2.setState({
                    notificationErrorMessage: "Failure occured fetching webhooks, error returned from the REST endpoint was : " + text,
                    notificationStatus: 'error',
                    notificationStatusMsgShort: 'Error:',
                    showNotification: true
                  });
                });
              case 10:
              case "end":
                return _context.stop();
            }
          }
        }, _callee, this, [[0, 7]]);
      }));
      function componentDidMount() {
        return _componentDidMount.apply(this, arguments);
      }
      return componentDidMount;
    }()
  }, {
    key: "formatCellContent",
    value: function formatCellContent(id, value) {
      if (id.endsWith(":repository")) {
        return React.createElement("a", {
          href: value,
          target: "_blank"
        }, value);
      } else {
        return value;
      }
    }
  }, {
    key: "render",
    value: function render() {
      var _this3 = this;
      if (this.state.isLoaded) {
        if (!this.state.webhooks.length) {
          return (
            React.createElement(ReactRouterDOM.Redirect, {
              to: this.props.match.url + "/create"
            })
          );
        } else {
          var headers = [{
            key: 'name',
            header: 'Name'
          }, {
            key: 'repository',
            header: 'Git Repository'
          }, {
            key: 'pipeline',
            header: 'Pipeline'
          }, {
            key: 'namespace',
            header: 'Namespace'
          }];
          var initialRows = [];
          this.state.webhooks.map(function (webhook, keyIndex) {
            initialRows[keyIndex] = {
              id: webhook['name'] + "|" + webhook['namespace'],
              name: webhook['name'],
              repository: webhook['gitrepositoryurl'],
              pipeline: webhook['pipeline'],
              namespace: webhook['namespace']
            };
          });
          return React.createElement("div", {
            className: "table-container"
          }, this.props.showNotificationOnTable && React.createElement(CarbonComponentsReact.InlineNotification, {
            kind: "success",
            subtitle: "",
            title: "Webhook created successfully."
          }), React.createElement(CarbonComponentsReact.DataTable, {
            rows: initialRows,
            headers: headers,
            render: function render(_ref) {
              var rows = _ref.rows,
                  headers = _ref.headers,
                  getHeaderProps = _ref.getHeaderProps,
                  getRowProps = _ref.getRowProps,
                  getSelectionProps = _ref.getSelectionProps,
                  getBatchActionProps = _ref.getBatchActionProps,
                  selectedRows = _ref.selectedRows,
                  onInputChange = _ref.onInputChange;
              return React.createElement(TableContainer, {
                title: "Webhooks"
              }, React.createElement(CarbonComponentsReact.TableToolbar, {
                id: "toolbar"
              }, React.createElement(CarbonComponentsReact.TableBatchActions, getBatchActionProps(), React.createElement(CarbonComponentsReact.TableBatchAction, {
                id: "delete-btn",
                renderIcon: _16,
                onClick: function onClick() {
                  _this3.handleSelectedRows(selectedRows);
                }
              }, "Delete")), React.createElement(CarbonComponentsReact.TableToolbarContent, null, React.createElement("div", {
                className: "search-bar"
              }, React.createElement(CarbonComponentsReact.TableToolbarSearch, {
                onChange: onInputChange
              })), React.createElement("div", {
                className: "btn-div"
              }, React.createElement(CarbonComponentsReact.Button, {
                as: ReactRouterDOM.Link,
                id: "create-btn",
                to: _this3.props.match.url + "/create"
              }, "Add", React.createElement("div", {
                className: "create-icon"
              }, React.createElement(_16$1, null)))))), React.createElement(Table, {
                className: "bx--data-table--zebra"
              }, React.createElement(TableHead, null, React.createElement(TableRow, null, React.createElement(CarbonComponentsReact.TableSelectAll, getSelectionProps()), headers.map(function (header) {
                return React.createElement(TableHeader, _extends({}, getHeaderProps({
                  header: header
                }), {
                  isSortable: "true",
                  isSortHeader: "true"
                }), header.header);
              }))), React.createElement(TableBody, null, rows.map(function (row) {
                return React.createElement(TableRow, _extends({}, getRowProps({
                  row: row
                }), {
                  key: row.id
                }), React.createElement(CarbonComponentsReact.TableSelectRow, getSelectionProps({
                  row: row
                })), row.cells.map(function (cell) {
                  return React.createElement(TableCell, {
                    key: cell.id
                  }, _this3.formatCellContent(cell.id, cell.value));
                }));
              }))));
            }
          }));
        }
      } else {
        if (this.state.showNotification) {
          return React.createElement("div", null, this.state.showNotification && React.createElement(CarbonComponentsReact.InlineNotification, {
            kind: this.state.notificationStatus,
            subtitle: this.state.notificationErrorMessage,
            title: this.state.notificationStatusMsgShort
          }));
        } else {
          return React.createElement("div", {
            className: "spinner-div"
          }, React.createElement(CarbonComponentsReact.Loading, {
            withOverlay: false,
            active: "true",
            className: "loading-spinner"
          }));
        }
      }
    }
  }]);
  return WebhookDisplayTable;
}(React.Component);

var WebhooksApp =
function (_Component) {
  _inherits(WebhooksApp, _Component);
  function WebhooksApp(props) {
    var _this;
    _classCallCheck(this, WebhooksApp);
    _this = _possibleConstructorReturn(this, _getPrototypeOf(WebhooksApp).call(this, props));
    _this.state = {
      showNotificationOnTable: false
    };
    _this.setShowNotificationOnTable = _this.setShowNotificationOnTable.bind(_assertThisInitialized(_this));
    return _this;
  }
  _createClass(WebhooksApp, [{
    key: "setShowNotificationOnTable",
    value: function setShowNotificationOnTable(value) {
      this.setState({
        showNotificationOnTable: value
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this2 = this;
      var match = this.props.match;
      return React.createElement("div", null, React.createElement(ReactRouterDOM.Route, {
        exact: true,
        path: "".concat(match.path, "/"),
        render: function render(props) {
          return React.createElement(WebhookDisplayTable, _extends({}, props, {
            showNotificationOnTable: _this2.state.showNotificationOnTable
          }));
        }
      }), React.createElement(ReactRouterDOM.Route, {
        path: "".concat(match.path, "/create"),
        render: function render(props) {
          return React.createElement(WebhookCreate, _extends({}, props, {
            setShowNotificationOnTable: _this2.setShowNotificationOnTable
          }));
        }
      }));
    }
  }]);
  return WebhooksApp;
}(React.Component);
var WebhookApp = ReactRouterDOM.withRouter(WebhooksApp);

export default WebhookApp;
