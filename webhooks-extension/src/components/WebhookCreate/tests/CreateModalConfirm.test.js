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

global.scrollTo = jest.fn();

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

const serviceAccountsResponseMock = {
  "items": [
    {
      "metadata": {
        "name": "default",
      },
    },
    {
      "metadata": {
        "name": "testserviceaccount",
      },
    }
  ]
}

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

  it('create should hide modal and return to form with new secret selected', async () => {
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    jest.spyOn(API, 'createSecret').mockImplementation((request) => {
      const expectRequest = { name: 'new-secret-foo', accesstoken: '1234567890bar' };
      expect(request).toStrictEqual(expectRequest);
      return Promise.resolve({});
    });

    const { getByText } = renderWithRouter(
      <WebhookCreate
        match={{}}
        namespaces={namespaces}
        pipelines={pipelines}
        setShowNotificationOnTable={() => {}}
        fetchPipelines={() => {}}
        isFetchingPipelines={false}
      />
    );
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    fireEvent.click(await waitForElement(() => document.getElementById('create-secret-button')));

    const name = document.getElementById('secretName')
    fireEvent.change(name, { target: { value: 'new-secret-foo' } });
    const token = document.getElementById('tokenValue')
    fireEvent.change(token, { target: { value: '1234567890bar' } });

    expect(document.getElementsByClassName('notification').item(0).childElementCount).toBe(0);
    fireEvent.click(document.getElementsByClassName('create-modal').item(0).getElementsByClassName('bx--btn--primary').item(0))
    await waitForElement(() => getByText(/Secret created/i));
    expect(document.getElementsByClassName('notification').item(0).childElementCount).toBe(1);
    expect(document.getElementById('git').getElementsByClassName('bx--list-box__label').item(0).textContent).toBe("new-secret-foo")
  });
  
})
