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
import { waitForElement, cleanup, fireEvent } from 'react-testing-library';
import WebhookApp from './WebhookApp'
import * as API from './api';
import { renderWithRouter } from './test/utils/test'
import 'react-testing-library/cleanup-after-each';

global.scrollTo = jest.fn();

beforeEach(() => {
  jest.restoreAllMocks
  jest.resetModules()
});

afterEach(() => {
  jest.clearAllMocks()
  cleanup()
});

function fakeDeleteWebhooksSuccess() {
  return {
    data: {},
    status: 204,
    statusText: 'OK',
    headers: {}
  };
}

const fakeRowSelection = [
  {
    "id":"mywebhook|default",
    "isSelected":true,
    "isExpanded":false,
    "cells":[
      {
        "id":"mywebhook|default:name","value":"mywebhook","isEditable":false,"isEditing":false,"isValid":true,"errors":null,"info":
        {
          "header":"name"
        }
      },
      {
        "id":"mywebhook|default:repository","value":"https://github.com/foo/bar","isEditable":false,"isEditing":false,"isValid":true,"errors":null,"info":
        {
          "header":"repository"
        }
      },
      {
        "id":"mywebhook|default:pipeline","value":"simple-helm-pipeline-insecure","isEditable":false,"isEditing":false,"isValid":true,"errors":null,"info":
        {
          "header":"pipeline"
        }
      },
      {"id":"mywebhook|default:namespace","value":"default","isEditable":false,"isEditing":false,"isValid":true,"errors":null,"info":
      {
        "header":"namespace"
      }
    }
  ]}
]

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

const webhooks = [
  {
    id: '0|namespace',
    name: 'first test webhook',
    gitrepositoryurl: 'repo1',
    pipeline: 'pipeline1',
    namespace: 'namespace1'
  }
];

it('change in components after last webhook deleted', async () => {
  let getWebhooksMock = jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
  let getRowsMock = jest.spyOn(API, "getSelectedRows").mockImplementation(() => fakeRowSelection);
  let deleteWebhooksMock = jest.spyOn(API, "deleteWebhooks").mockImplementation(() => Promise.resolve(fakeDeleteWebhooksSuccess));

  const { getByText, queryByTestId } = renderWithRouter(<WebhookApp match={{}} />);

  await waitForElement(() => getByText('first test webhook'));
  expect(queryByTestId('table-container')).not.toBeNull();
  expect(queryByTestId('webhook-create')).toBeNull();

  const foundDeleteButton = document.getElementById('delete-btn');
  await waitForElement(() => foundDeleteButton);

  fireEvent.click(foundDeleteButton);

  const foundDeleteButtonOnModal = document.getElementById('webhook-delete-modal').getElementsByClassName('bx--btn bx--btn--danger').item(0);
  await waitForElement(() => foundDeleteButtonOnModal);

  fireEvent.click(foundDeleteButtonOnModal);

  expect(getWebhooksMock).toHaveBeenCalled();
  expect(getRowsMock).toHaveBeenCalled();
  expect(deleteWebhooksMock).toHaveBeenCalled();

  getWebhooksMock.mockImplementation(() => Promise.resolve([]));
  jest.spyOn(API, 'getNamespaces').mockImplementation(() => Promise.resolve(namespacesResponseMock));
  jest.spyOn(API, 'getPipelines').mockImplementation(() => Promise.resolve(pipelinesResponseMock));
  jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));
  jest.spyOn(API, 'getServiceAccounts').mockImplementation(() => Promise.resolve(serviceAccountsResponseMock));

  await waitForElement(() => getByText('Last webhook deleted successfully.'));
  expect(queryByTestId('table-container')).toBeNull();
  expect(queryByTestId('webhook-create')).not.toBeNull();
});
