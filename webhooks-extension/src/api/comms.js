/*
Copyright 2019-2020 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

const CSRF_HEADER = 'X-CSRF-Token';
const CSRF_SAFE_METHODS = ['GET', 'HEAD', 'OPTIONS'];

const defaultOptions = {
  method: 'GET'
};

const apiRoot = getDashboardAPIRoot();

export function getDashboardAPIRoot() {
  const { href, hash } = window.location;
  let baseURL = href.replace(hash, '');
  if (baseURL.endsWith('/')) {
    baseURL = baseURL.slice(0, -1);
  }
  return baseURL;
}

function getToken() {
  return fetch(`${apiRoot}/v1/token`, {
    ...defaultOptions,
    headers: {
      Accept: 'text/plain'
    }
  }).then(response => response.headers.get(CSRF_HEADER));
}

export function getHeaders(headers = {}) {
  return {
    Accept: 'application/json',
    'Content-Type': 'application/json',
    ...headers
  };
}

export function checkStatus(response = {}) {
  if (response.ok) {
    switch (response.status) {
      case 201:
        return response.headers;
      case 204:
        return {};
      default: {
        let responseAsJson = response.json();
        return responseAsJson;
      }
    }
  }

  const error = new Error(response.statusText);
  error.response = response;
  throw error;
}

export async function request(uri, options = defaultOptions) {
  let token;
  if (!CSRF_SAFE_METHODS.includes(options.method)) {
    token = await getToken();
  }

  const headers = {
    ...options.headers,
    ...(token && { [CSRF_HEADER]: token })
  };

  return fetch(uri, {
    ...defaultOptions,
    ...options,
    headers
  }).then(checkStatus);
}

export function get(uri) {
  return request(uri, {
    method: 'GET',
    headers: getHeaders()
  });
}

export function post(uri, body) {
  return request(uri, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(body)
  });
}

export function put(uri, body) {
  return request(uri, {
    method: 'PUT',
    headers: getHeaders(),
    body: JSON.stringify(body)
  });
}

export function deleteRequest(uri) {
  return request(uri, {
    method: 'DELETE',
    headers: getHeaders()
  });
}
