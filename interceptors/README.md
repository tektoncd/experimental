# Experimental/Advanced Interceptors

This project demonstrates a number of additional interceptors that can be deployed to a Tekton Triggers
installation. It is intended to demonstrate the capabilities of the `interceptor-sdk` and provide
Triggers users with more capabilities around validating the data sent to an EventListener endpoint.

## Interceptors

This project demonstrates a number of custom interceptors:

* [`hmac`](#hmac)
* [`jwt`](#jwt)
* [`rego`](#rego)

These three interceptors do not currently set any extensions, but this could be added at a later point.

### HMAC

The HMAC interceptor validates an HMAC signature provided in an HTTP request header. The signature is
calculated using the HTTP request body and a secret key that is provided in the configuration of the interceptor.

```YAML
- ref:
    name: "hmac"
  params:
  - name: "secretRef"
    value:
      secretName: foo
      secretKey: bar
  - name: "header"
    value: "X-Signature"
```

The interceptor expects an algorithm to be encoded in the header as `<algorithm>=<value>`. If the algorithm is
not provided in the header value, then you can provide the algorithm to the interceptor as a parameter.

```YAML
- ref:
    name: "hmac"
  params:
  - name: "secretRef"
    value:
      secretName: foo
      secretKey: bar
  - name: "header"
    value: "X-Signature"
  - name: "algorithm"
    value: "sha1"
```

The supported algorithms are `sha1`, `sha256`, and `sha512`.

### JWT

The JWT token interceptor expected a JWT to be provided in an HTTP request header and validates token
data against specific claims. If a JWKS URI parameter is set, the JWT will be cryptographically validated.
The token is also validated against the provided expiration time.

The JWT token is extracted from a Bearer Authorization header, that is of the format `Authorization: Bearer <token>`. If other formats are required for this filter, please provide direction for how this would be extracted from the EventListener request.

```YAML
- ref:
    name: "jwt"
  params:
  - name: "audience"
    value: "tekton"
  - name: "issuer"
    value: "my-issuer"
  - name: jwks_url
    value: https://my-jwks-url.my-org.com
```

All of the parameters in the JWT interceptor are optional, so if no parameters are specified it just validates
that the token is valid and not expired.

### Rego

The rego interceptor expects you to provide a module and a query to validate against the input body, which is assembled from the interceptor request. The rego input data contains three fields that you can reference:

* `input.body`
* `input.extensions`
* `input.header`

The rego query expects to return a single boolean value to validate whether interceptor processing should continue.

```YAML
- ref:
    name: "rego"
  params:
  - name: "module"
    value: |
      package tekton

      default allow = false

      allow {
        input.body.value = "test"
      }
  - name: "query"
    value: >
      data.rego.allow
```


