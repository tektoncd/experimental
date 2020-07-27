import React from 'react';
import '@patternfly/react-core/dist/styles/base.css';

import {
  Tabs,
  Tab,
  Card,
  Grid,
  GridItem,
  CardHead,
} from '@patternfly/react-core';
import './index.css';
import ReactMarkDown from 'react-markdown';
import CodeBlock from './CodeBlock';
import CodeBlockReadme from './CodeBlockReadme';
import Loader from '../loader/loader';

export interface DescriptionProp {
  // id: any
  Description: string,
  Yaml: string,
  userTaskDescription: string
}


const Description: React.FC<DescriptionProp> = (props: any) => {
  const [activeTabKey, setActiveTabKey] = React.useState(0);
  const handleTabClick = (event: any, tabIndex: any) => {
    setActiveTabKey(tabIndex);
  };


  let markDown: string = '';
  if (props.Description != null) {
    if (props.Description.match('noreadme')) {
      markDown = props.userTaskDescription;
    } else {
      markDown = props.Description;
    }
  }

  let markDownYaml: string = '';
  if (props.Yaml != null) {
    if (props.Yaml.match('noyaml')) {
      markDownYaml = 'YAML file not found';
    } else {
      markDownYaml = props.Yaml;
    }
  }

  if (props.Description === undefined) {
    return (
      <Loader />
    );
  }

  return (
    <Grid>
      <GridItem span={11} className="pf-u-ml-sm-on-lg">
        <Card style={{marginLeft: '9em', marginRight: '2em'}}>
          <CardHead style={{paddingTop: '2em'}}>
            <Grid style={{width: '90em'}}>
              <GridItem span={12}>
                <Tabs activeKey={activeTabKey} isSecondary
                  onSelect={handleTabClick} style={{boxShadow: 'none'}}>
                  <Tab eventKey={0} title="Description"
                    style={{backgroundColor: 'white'}}>
                    <hr
                      style={{
                        backgroundColor: '#EDEDED',
                        marginBottom: '1em',
                      }}>
                    </hr>
                    <ReactMarkDown source={markDown}
                      escapeHtml={true}
                      renderers={{code: CodeBlockReadme}}
                      className="readme"
                    />
                  </Tab>
                  <Tab eventKey={1} title="YAML"
                    style={{backgroundColor: 'white'}}>
                    <hr
                      style={{
                        backgroundColor: '#EDEDED',
                        marginBottom: '1em',
                      }}>
                    </hr>
                    <ReactMarkDown source={markDownYaml}
                      escapeHtml={true}
                      renderers={{code: CodeBlock}}
                      className="yaml"
                    />
                  </Tab>
                </Tabs>
              </GridItem>
            </Grid>
          </CardHead>
        </Card>
      </GridItem>
    </Grid>
  );
};

export default Description;
