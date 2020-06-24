import React from 'react';
import {
  Card,
  CardHead,
  CardActions,
  CardFooter,
  TextContent,
  Text,
  GridItem,
  Grid,
  Flex,
  FlexItem,
  TextVariants,
} from '@patternfly/react-core';
import tekton from '../assets/logo/logo.png';

const Footer: React.FC = () => {
  return (

    <div>
      <Card style={{height: '20em', backgroundColor: '#EDEDED'}}></Card>
      <Card style={{backgroundColor: '#151515'}}>
        <CardHead>
          <img src={tekton} alt="Task"
            style={{height: '7em', marginLeft: '5em'}}
          />
          <TextContent style={{marginLeft: '2em', color: 'white'}}>
            <Text component={TextVariants.h1}>
              Tekton
            </Text>
            <Grid>
              <GridItem span={6}>
                <Text style={{color: 'white'}}>
                  The Tekton Pipelines project
                  provides k8s-style resources for declaring
                  CI/CD-style pipelines.Click here to learn more about
                  <a href="https://github.com/tektoncd/pipeline" target="_">Tekton</a>
                </Text>
              </GridItem>
            </Grid>
          </TextContent>

          <CardActions style={{marginTop: '1.5em'}}>
            <Flex breakpointMods={[{modifier: 'nowrap', breakpoint: 'lg'}]}>

              <FlexItem style={{marginRight: '5em'}}>
                <Flex breakpointMods
                  ={[{modifier: 'column', breakpoint: 'lg'}]}>

                  <FlexItem >
                    <TextContent>
                      <Text style={{color: 'white', marginTop: '1.5em'}}
                        component={TextVariants.h1}>
                        Links
                      </Text>

                      <Text>
                        <a href="/"
                          style={{color: 'white', marginTop: '1.5em'}} >
                          About
                        </a>
                      </Text>

                      <Text>
                        <a href="/"
                          style={{color: 'white', marginTop: '1.5em'}}>
                          Contribute
                        </a>
                      </Text>

                      <Text>
                        <a href="/"
                          style={{color: 'white', marginTop: '1.5em'}}>
                          Tekton
                        </a>
                      </Text>

                    </TextContent>
                  </FlexItem>
                </Flex>
              </FlexItem>

              <FlexItem style={{marginRight: '10em'}}>
                <TextContent>
                  <Text style={{color: 'white', marginTop: '1.5em'}}
                    component={TextVariants.h1}>
                    Contribute
                  </Text>
                  <Text>
                    <a href="/"
                      style={{color: 'white', marginTop: '1.5em'}}>
                      About
                    </a>
                  </Text>

                  <Text>
                    <a href="/"
                      style={{color: 'white', marginTop: '1.5em'}}>
                      Contribute
                    </a>
                  </Text>

                  <Text>
                    <a href="/"
                      style={{color: 'white', marginTop: '1.5em'}}>
                      Tekton
                    </a>
                  </Text>

                </TextContent>
              </FlexItem>
            </Flex>

          </CardActions>
        </CardHead>

        <CardFooter style={{marginLeft: '45%'}}>
          <Text style={{color: 'white'}}>
            Copyright © 2019 Red Hat, Inc.
          </Text>
        </CardFooter>

      </Card>
    </div>
  );
};

export default Footer;
