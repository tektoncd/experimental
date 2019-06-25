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


const namespacesResponseMock = {
  "items": [
    {
      "metadata": {
        "name": "default",
      }
    },
    {
      "metadata": {
        "name": "docker",
      }
    },
    {
      "metadata": {
        "name": "istio-system",
      },
    },
    {
      "metadata": {
        "name": "knative-eventing",
      },
    }
  ]
};

const pipelinesResponseMock = {
  "items": [
    {
      "metadata": {
        "name": "simple-helm-pipeline",
      }
    },
    {
      "metadata": {
        "name": "simple-helm-pipeline-insecure",
      }
    },
  ]
}

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
describe('delete and create secret', () => {
  it('buttons should be disabled until namespace selected', async () => {   
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />); 
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    await waitForElement(() => document.getElementsByClassName('del-sec-btn'));

    expect(document.getElementById('delete-secret-button').getAttribute('class')).toBe('secButtonDisabled');
    expect(document.getElementById('create-secret-button').getAttribute('class')).toBe('secButtonDisabled');

    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    await waitForElement(() => document.getElementsByClassName('secButtonEnabled'));

    expect(document.getElementById('delete-secret-button').getAttribute('class')).toBe('secButtonEnabled');
    expect(document.getElementById('create-secret-button').getAttribute('class')).toBe('secButtonEnabled');
    });
})

//-----------------------------------//
describe('delete secret', () => {

  it('should display error notification if no secret selected and delete pressed', async () => {   
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />); 
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));

    await waitForElement(() => document.getElementsByClassName('secButtonEnabled'));
    fireEvent.click(document.getElementById('delete-secret-button'));
    await waitForElement(() => getByText(/No secret selected. A secret must be selected from the drop down before selecting delete./));

  });

  it('modal should be shown to confirm deletion and include selected secret name', async () => {  
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />); 
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    fireEvent.click(await waitForElement(() => getByText(/select secret/i)));
    fireEvent.click(await waitForElement(() => getByText(/ghe/i)));

    expect(document.getElementById('delete-modal').getAttribute('class')).not.toContain('is-visible');
    fireEvent.click(document.getElementById('delete-secret-button'));
    expect(document.getElementById('delete-modal').getAttribute('class')).toContain('is-visible');

  });
  
  it('cancel button should hide modal', async () => {

    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    await waitForElement(() => document.getElementsByClassName('secButtonEnabled'));
    fireEvent.click(await waitForElement(() => getByText(/select secret/i)));
    fireEvent.click(await waitForElement(() => getByText(/ghe/i)));
    await waitForElement(() => document.getElementsByClassName('secButtonEnabled'));
    fireEvent.click(document.getElementById('delete-secret-button'));
    expect(document.getElementById('delete-modal').getAttribute('class')).toContain('is-visible');

    fireEvent.click(document.getElementById('delete-modal').getElementsByClassName('bx--btn--secondary').item(0));
    expect(document.getElementById('delete-modal').getAttribute('class')).not.toContain('is-visible');
    await waitForElement(() => getByText(/istio-system/i))

  });

})
