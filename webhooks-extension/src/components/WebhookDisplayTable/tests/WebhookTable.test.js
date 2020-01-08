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
import { waitForElement, fireEvent } from 'react-testing-library';
import { WebhookDisplayTable } from '../WebhookDisplayTable'
import * as API from '../../../../src/api/index';
import { renderWithRouter } from '../../../test/utils/test'
import 'react-testing-library/cleanup-after-each';

beforeEach(jest.restoreAllMocks);

describe('without webhooks', () => {
  const noWebhooks = [ {} ];
  it('should display Loading when loading', () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve([noWebhooks]));
    const { queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*"/>);
    expect(queryByTestId('webhook-notification')).toBeNull();
    expect(queryByTestId("loading-table")).toBeTruthy();
  });
});

describe('displays an add button', () => {
  const noWebhooks = [ {} ];
  it('display an add button for creating webhooks', async () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve([noWebhooks]));
    const { getByText, queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*"/>);
    expect(queryByTestId('webhook-notification')).toBeNull();
    await waitForElement(() => getByText(/Add/i));
   });
});

describe('with webhooks', () => {
  const webhooks = [
    {
      id: '0|namespace',
      name: 'first-test-webhook',
      gitrepositoryurl: 'the-webhook-repo',
      pipeline: 'the-pipeline',
      serviceaccount: 'the-service-account',
      namespace: 'webhook-namespace'
    }
  ];

  it('displays webhooks table when webhooks are present', async () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve(webhooks));
    const { getByText, queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*"/>);
    expect(queryByTestId('webhook-notification')).toBeNull();
    await waitForElement(() => getByText(/webhooks/i)); // Webhooks header (table visible) only when webhooks present
  });

  it('displays webhooks table and data webhooks are present', async () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve(webhooks));
    const { getByText, queryByTestId } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="*"/>);
    expect(queryByTestId('webhook-notification')).toBeNull();
    await waitForElement(() => getByText(/first-test-webhook/i));
    await waitForElement(() => getByText(/the-webhook-repo/i));
    await waitForElement(() => getByText(/the-pipeline/i));
    await waitForElement(() => getByText(/the-service-account/i));
    await waitForElement(() => getByText(/webhook-namespace/i));
  });

  it('displays webhook created successfully on show notification being set', async () => {
    const props = {
      showNotificationOnTable: true,
      match: {
        url: "/"
      }
    }
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve(webhooks));
    const { getByText } = renderWithRouter(<WebhookDisplayTable {...props} selectedNamespace="*"/>);
    await waitForElement(() => getByText('Webhook created successfully.'));
  });

  it('should display string suggesting to create webhooks when namespace selected has no webhooks', async () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve(webhooks));
    const { getByText } = renderWithRouter(<WebhookDisplayTable match={{}} selectedNamespace="default"/>);
    await waitForElement(() => getByText("No webhooks created under namespace 'default', click 'Add Webhook' button to add a new one."));
  });

  it('display branch modal when table row clicked', async () => {
    jest.spyOn(API, 'getWebhooks').mockImplementation(() => Promise.resolve(webhooks));
    const { getByText } = renderWithRouter(<WebhookDisplayTable fetchPipelineRuns={jest.fn(() => Promise.resolve([]))} match={{}} selectedNamespace="*"/>);

    await waitForElement(() => getByText(/first-test-webhook/i));

    fireEvent.click(getByText(/first-test-webhook/i));
    expect(getByText("Latest PipelineRuns By Branch:")).not.toBeNull();

  });
});
