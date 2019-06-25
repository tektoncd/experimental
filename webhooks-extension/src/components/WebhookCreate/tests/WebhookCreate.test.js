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
import { waitForElement, wait, fireEvent, cleanup } from 'react-testing-library';
import { renderWithRouter } from '../../../test/utils/test';
import 'react-testing-library/cleanup-after-each';
import { WebhookCreate } from '../WebhookCreate';
import * as API from '../../../api/index';
import 'jest-dom/extend-expect'

window.scrollTo = jest.fn();

const namespacesFailResponseMock = {
  response: {
    text() {
      return Promise.resolve("Mock Error Fetching Namespaces")
    }
  }
};

const WebhookCreationFailMock = {
  response: {
    text() {
      return Promise.resolve("Mock Error Creating Webhook")
    }
  }
};

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
describe('error fetching namespaces should give error notification', () => {
  it('should display create wizard with form properties and error notification', async () => {   
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.reject(namespacesFailResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}}/>); 
      await waitForElement(() => getByText(/Mock Error Fetching Namespaces/i));
    });
})

//-----------------------------------//
describe('drop downs should be disabled while no namespace selected', () => {
  it('pipelines dropdown should be disabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    expect(document.getElementById("pipeline").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(true)

  });
  it('secrets dropdown should be disabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    expect(document.getElementById("git").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(true)

  });
  it('service accounts dropdown should be disabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    expect(document.getElementById("serviceAccounts").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(true)

    });
})

//-----------------------------------//
describe('drop downs should be enabled when a namespace is selected', () => {
  it('pipelines dropdown should be enabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)))
    await wait(() => expect(document.getElementById("pipeline").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(false))
  });
  it('secrets dropdown should be enabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)))
    await wait(() => expect(document.getElementById("git").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(false))

  });
  it('service accounts dropdown should be enabled', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)))
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)))
    await wait(() => expect(document.getElementById("serviceAccounts").getElementsByClassName("bx--list-box__field").item(0).hasAttribute("disabled")).toBe(false))

    });
})

//-----------------------------------//
describe('create button enablement', () => {
  // Increase timeout as lots involved in this test
  jest.setTimeout(7500);
  it('create button should be enabled only when all fields complete', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    const { getByText, getByTestId } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    
    fireEvent.click(await waitForElement(() => getByTestId('display-name-entry')));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    
    const name = getByTestId('display-name-entry')
    fireEvent.change(name, { target: { value: 'test-webhook-name' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('git-url-entry')));
    const gitUrl = getByTestId('git-url-entry')
    fireEvent.change(gitUrl, { target: { value: 'some.url.here' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('docker-reg-entry')));
    const dockerReg = getByTestId('docker-reg-entry')
    fireEvent.change(dockerReg, { target: { value: 'somevalue' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select pipeline/i)));
    fireEvent.click(await waitForElement(() => getByText(/simple-helm-pipeline-insecure/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select secret/i)));
    fireEvent.click(await waitForElement(() => getByText(/ghe/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select service account/i)));
    fireEvent.click(await waitForElement(() => getByText(/testServiceAccount/i)))

    // All fields set, button should be enabled
    await wait(() => expect(getByTestId('create-button')).toBeEnabled())

    // Start toggling text fields to check unsetting any text field disables the button
    fireEvent.change(name, { target: { value: '' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    fireEvent.change(name, { target: { value: 'test-webhook-name' } });
    await wait(() => expect(getByTestId('create-button')).toBeEnabled())

    fireEvent.change(gitUrl, { target: { value: '' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    fireEvent.change(gitUrl, { target: { value: 'some.url.here' } });
    await wait(() => expect(getByTestId('create-button')).toBeEnabled())

    fireEvent.change(dockerReg, { target: { value: '' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    fireEvent.change(dockerReg, { target: { value: 'somevalue' } });
    await wait(() => expect(getByTestId('create-button')).toBeEnabled())
  });
})

//-----------------------------------//
describe('create button', () => {
  // Increase timeout as lots involved in this test
  jest.setTimeout(7500);
  it('create failure should return notification', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    jest.spyOn(API, 'createWebhook').mockImplementation(() => Promise.reject(WebhookCreationFailMock));
    const { getByText, getByTestId } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => { }} />);
    
    fireEvent.click(await waitForElement(() => getByTestId('display-name-entry')));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    
    const name = getByTestId('display-name-entry')
    fireEvent.change(name, { target: { value: 'test-webhook-name' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('git-url-entry')));
    const gitUrl = getByTestId('git-url-entry')
    fireEvent.change(gitUrl, {target: { value: 'some.url.here'}});
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('docker-reg-entry')));
    const dockerReg = getByTestId('docker-reg-entry')
    fireEvent.change(dockerReg, { target: { value: 'somevalue' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select pipeline/i)));
    fireEvent.click(await waitForElement(() => getByText(/simple-helm-pipeline-insecure/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select secret/i)));
    fireEvent.click(await waitForElement(() => getByText(/ghe/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select service account/i)));
    fireEvent.click(await waitForElement(() => getByText(/testServiceAccount/i)));

    expect(document.getElementsByClassName('notification').item(0).childElementCount).toBe(0);
    await wait(() => expect(getByTestId('create-button')).toBeEnabled());
    fireEvent.click(document.getElementById('submit'));
    await waitForElement(() => getByText(/Mock Error Creating Webhook/i));
    expect(document.getElementsByClassName('notification').item(0).childElementCount).toBe(1)
  })

  it('success should set showNotification and call returnToTable', async () => {
    jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
    jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
    jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));
    jest.spyOn(API, 'createWebhook').mockImplementation((request) => {
      const expected = {
        name: 'test-webhook-name',
        gitrepositoryurl: 'some.url.here',
        accesstoken: 'ghe',
        pipeline: 'simple-helm-pipeline-insecure',
        namespace: 'istio-system',
        serviceaccount: 'testserviceaccount',
        dockerregistry: 'somevalue'
      }
      expect(request).toStrictEqual(expected);
      return Promise.resolve({})
    });

    const { getByText, getByTestId } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    
    fireEvent.click(await waitForElement(() => getByTestId('display-name-entry')));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())
    
    const name = getByTestId('display-name-entry')
    fireEvent.change(name, { target: { value: 'test-webhook-name' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('git-url-entry')));
    const gitUrl = getByTestId('git-url-entry')
    fireEvent.change(gitUrl, {target: { value: 'some.url.here'}});
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByTestId('docker-reg-entry')));
    const dockerReg = getByTestId('docker-reg-entry')
    fireEvent.change(dockerReg, { target: { value: 'somevalue' } });
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select namespace/i)));
    fireEvent.click(await waitForElement(() => getByText(/istio-system/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select pipeline/i)));
    fireEvent.click(await waitForElement(() => getByText(/simple-helm-pipeline-insecure/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select secret/i)));
    fireEvent.click(await waitForElement(() => getByText(/ghe/i)));
    await wait(() => expect(getByTestId('create-button')).toBeDisabled())

    fireEvent.click(await waitForElement(() => getByText(/select service account/i)));
    fireEvent.click(await waitForElement(() => getByText(/testServiceAccount/i)));

    expect(document.getElementsByClassName('notification').item(0).childElementCount).toBe(0);
    await wait(() => expect(getByTestId('create-button')).toBeEnabled());
    fireEvent.click(document.getElementById('submit'));

  })
})

describe('tooltips', () => {
  it('click on name tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('name-tooltip'))
    fireEvent.click(document.getElementById('name-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The display name for your webhook in this user interface./i))
  });

  it('click on git url tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('git-tooltip'))
    fireEvent.click(document.getElementById('git-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The URL of the git repository to create the webhook on./i))
  });
  it('click on pipeline tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('pipeline-tooltip'))
    fireEvent.click(document.getElementById('pipeline-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The pipeline from the selected namespace to run when the webhook is triggered./i))
  });
  it('click on namespace tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('namespace-tooltip'))
    fireEvent.click(document.getElementById('namespace-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The namespace to operate in./i))
  });
  it('click on secrets tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('secret-tooltip'))
    fireEvent.click(document.getElementById('secret-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The kubernetes secret holding access information for the git repository. The credential must have sufficient privileges to create webhooks in the repository./i))
  });
  it('click on service account tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('serviceaccount-tooltip'))
    fireEvent.click(document.getElementById('serviceaccount-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The service account under which to run the pipeline run./i))
  });
  it('click on docker registry tooltip', async () => {
    const { getByText } = renderWithRouter(<WebhookCreate match={{}} setShowNotificationOnTable={() => {}} />);
    await waitForElement(() => document.getElementById('docker-tooltip'))
    fireEvent.click(document.getElementById('docker-tooltip').getElementsByClassName('bx--tooltip__trigger').item(0))
    await waitForElement(() => getByText(/The docker registry to push images to./i))
  });
})