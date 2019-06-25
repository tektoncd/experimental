/*
Copyright 2019 The Tekton Authors
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

import { get, post, deleteRequest } from './comms';

const apiRoot = getAPIRoot();
const dashboardAPIRoot = getDashboardAPIRoot();

// Defined here for ease of mocking to test: would be great to remove
// perhaps using enzyme and mocking state?
export function getSelectedRows(selectedRows) {
  return selectedRows
}

export function getAPIRoot() {
  const { href, hash } = window.location;
  let newHash = hash.replace('#/extensions','v1/extensions')
  let baseURL = href.replace(hash, newHash);
  if (baseURL.endsWith('/')) {
    baseURL = baseURL.slice(0, -1);
  }
  return baseURL;
}

export function getDashboardAPIRoot() {
  const { href, hash } = window.location;
  let newHash = hash.replace('#/extensions/webhooks-extension','v1')
  let baseURL = href.replace(hash, newHash);
  if (baseURL.endsWith('/')) {
    baseURL = baseURL.slice(0, -1);
  }
  return baseURL;
}

export function getWebhooks() {
  const uri = `${apiRoot}/webhooks`;
  return get(uri);
}

export function createWebhook(data) {
  const uri = `${apiRoot}/webhooks`;
  return post(uri, data);
}

export function getSecrets(namespace) {
  const uri = `${apiRoot}/webhooks/credentials?namespace=${namespace}`;
  return get(uri);
}

export function createSecret(data, namespace) {
  const uri = `${apiRoot}/webhooks/credentials?namespace=${namespace}`;
  return post(uri, data);
}

export function deleteSecret(name, namespace) {
  const uri = `${apiRoot}/webhooks/credentials/${name}?namespace=${namespace}`;
  return deleteRequest(uri);
}

export function getNamespaces() {
  const uri = `${dashboardAPIRoot}/namespaces`;
  return get(uri);
}

export function getPipelines(namespace) {
  const uri = `${dashboardAPIRoot}/namespaces/${namespace}/pipelines`;
  return get(uri);
}

export function getServiceAccounts(namespace) {
  const uri = `${dashboardAPIRoot}/namespaces/${namespace}/serviceaccounts`;
  return get(uri);
}

export function deleteWebhooks(id, namespace, deleteRuns) {
  let deleteRunsQuery = ""
  if (deleteRuns) {
    deleteRunsQuery = "&deletepipelineruns=true";
  }
  const uri = `${apiRoot}/webhooks/${id}?namespace=${namespace}${deleteRunsQuery}`;
  return deleteRequest(uri);
}
