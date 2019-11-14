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
import { Provider } from 'react-redux';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';
import { waitForElement, cleanup, fireEvent } from 'react-testing-library';
import WebhookApp from './WebhookApp'
import * as API from './api';
import { renderWithRouter } from './test/utils/test'
import 'react-testing-library/cleanup-after-each';

const middleware = [thunk];
const mockStore = configureStore(middleware);

global.scrollTo = jest.fn();

beforeEach(() => {
  jest.restoreAllMocks
  jest.resetModules()
});

afterEach(() => {
  jest.clearAllMocks()
  cleanup()
});

const namespaces = {
  byName: {
    default: {
      metadata: {
        name: 'default',
        uid: '32b35d3b-6ce1-11e9-af21-025000000001',
      },
    }
  },
  errorMessage: null,
  isFetching: false,
  selected: 'default'
};

let store = mockStore({
  namespaces
});

function fakeDeleteWebhooksSuccess() {
  return {
    data: {},
    status: 204,
    statusText: 'OK',
    headers: {}
  };
}

const secretsResponseMock = [
  {
    "name": "ghe",
  },
  {
    "name": "git",
  }
]

const webhooks = [
  {
    id: '0|namespace',
    name: 'first test webhook',
    gitrepositoryurl: 'repo1',
    pipeline: 'pipeline1',
    namespace: 'default'
  },
  {
    id: '1|namespace',
    name: 'second test webhook',
    gitrepositoryurl: 'repo2',
    pipeline: 'pipeline1',
    namespace: 'default'
  },
  {
    id: '2|namespace',
    name: 'third test webhook',
    gitrepositoryurl: 'repo3',
    pipeline: 'pipeline1',
    namespace: 'default'
  }
];

const selectors = {
  getSelectedNamespace: jest.fn(() => "default"),
  getNamespaces: jest.fn(() => ["default", "namespace2", "namespace3"]),
  getPipelines: jest.fn(() => [
    {
      metadata: {
        name: "pipeline0",
        namespace: "default"
      }
    },
    {
      metadata: {
        name: "simple-pipeline",
        namespace: "default"
      }
    }
  ]),
  isFetchingNamespaces: jest.fn(() => false),
  isFetchingPipelines: jest.fn(() => false),
  getPipelinesErrorMessage: jest.fn(() => null),
  getServiceAccountsErrorMessage: jest.fn(() => null),
  isFetchingServiceAccounts: jest.fn(() => false),
  getServiceAccounts: jest.fn(() => [
    {
      metadata: {
        name: "default",
        namespace: "default"
      }
    },
    {
      metadata: {
        name: "second-sa",
        namespace: "default",
      },
    }
  ])
};

describe("change in components after last webhook(s) deleted & shows notification", () => {
  const test = async (webhooks) => {
    let getWebhooksMock = jest.spyOn(API, "getWebhooks").mockImplementation(() => Promise.resolve(webhooks));
    let deleteWebhooksMock = jest.spyOn(API, "deleteWebhooks").mockImplementation(() => Promise.resolve(fakeDeleteWebhooksSuccess));

    const { getByText, queryByTestId } = renderWithRouter(
      <Provider store={store}>
        <WebhookApp match={{}} selectors={selectors}/>
      </Provider>);


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
    expect(deleteWebhooksMock).toHaveBeenCalled();

    getWebhooksMock.mockImplementation(() => Promise.resolve([]));
    jest.spyOn(API, 'getSecrets').mockImplementation(() => Promise.resolve(secretsResponseMock));

    await waitForElement(() => getByText('Last webhook(s) deleted successfully.'));
    expect(queryByTestId('table-container')).toBeNull();
    expect(queryByTestId('webhook-create')).not.toBeNull();
  }

  it('# of webhooks: 1', () => {
    test([webhooks[0]]);
  });

  it('# of webhooks: 2', () => {
    test(webhooks.slice(0,2));
  });

  it('# of webhooks: 3', () => {
    test(webhooks);
  });
});
