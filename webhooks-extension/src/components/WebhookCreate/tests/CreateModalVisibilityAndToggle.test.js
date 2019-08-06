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

import React from 'react';
import { waitForElement, fireEvent, cleanup } from 'react-testing-library';
import { renderWithRouter } from '../../../test/utils/test';
import 'react-testing-library/cleanup-after-each';
import { WebhookCreate } from '../WebhookCreate';
import * as API from '../../../api/index';
import 'jest-dom/extend-expect'


const namespaces = ["default", "istio-system", "namespace3"];

const pipelines = [
  {
    metadata: {
      name: "pipeline0",
      namespace: "default",
    },
  },
  {
    metadata: {
      name: "simple-pipeline",
      namespace: "default",
    },
  },
  {
    metadata: {
      name: "simple-helm-pipeline-insecure",
      namespace: "istio-system",
    }
  }
];

const secretsResponseMock = [
  {
    "name": "ghe",
  },
  {
    "name": "git",
  }
]

const serviceAccounts = [
  {
    metadata: {
      name: "default",
      namespace: "default"
    }
  },
  {
    metadata: {
      name: "testserviceaccount",
      namespace: "istio-system",
    },
  }
];

beforeEach(() => {
  jest.restoreAllMocks
  jest.resetModules()
 });

afterEach(() => {
  jest.clearAllMocks()
  cleanup()
 });


//-----------------------------------//
describe('create secret', () => {

  it('create button enabled only when name and token complete', async () => {
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    const { getByText } = renderWithRouter(
      <WebhookCreate
        match={{}}
        namespaces={namespaces}
        pipelines={pipelines}
        fetchPipelines={() => {}}
        serviceAccounts={serviceAccounts}
        fetchServiceAccounts={() => {}}
        setShowNotificationOnTable={() => {}}
      />
    );
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));

    expect(document.getElementById('create-modal').getAttribute('class')).not.toContain('is-visible');
    fireEvent.click(document.getElementById('create-secret-button'));
    expect(document.getElementById('create-modal').getAttribute('class')).toContain('is-visible');
    expect(document.getElementsByClassName('create-modal').item(0).getElementsByClassName('bx--btn--primary').item(0).getAttributeNames()).toContain('disabled')

    const name = document.getElementById('secretName')
    fireEvent.change(name, { target: { value: 'new-secret-foo' } });
    expect(document.getElementsByClassName('create-modal').item(0).getElementsByClassName('bx--btn--primary').item(0).getAttributeNames()).toContain('disabled')

    const token = document.getElementById('tokenValue')
    fireEvent.change(token, { target: { value: '1234567890bar' } });
    expect(document.getElementsByClassName('create-modal').item(0).getElementsByClassName('bx--btn--primary').item(0).getAttributeNames()).not.toContain('disabled')

    fireEvent.change(name, { target: { value: '' } });
    expect(document.getElementsByClassName('create-modal').item(0).getElementsByClassName('bx--btn--primary').item(0).getAttributeNames()).toContain('disabled')

  });

  it('should be able toggle visibility of token', async () => {
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    const { getByText, getByLabelText } = renderWithRouter(
      <WebhookCreate
        match={{}}
        namespaces={namespaces}
        pipelines={pipelines}
        fetchPipelines={() => {}}
        serviceAccounts={serviceAccounts}
        fetchServiceAccounts={() => {}}
        setShowNotificationOnTable={() => {}}
      />
    );
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));

    const tokenInput = document.getElementById('tokenValue');

    expect(tokenInput.getAttribute('type')).toBe('password')

    fireEvent.click(getByLabelText('Show password'));

    expect(tokenInput.getAttribute('type')).toBe('text')

    fireEvent.click(getByLabelText('Hide password'));

    expect(tokenInput.getAttribute('type')).toBe('password')


  });

})
