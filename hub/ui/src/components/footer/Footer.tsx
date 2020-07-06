import React from 'react';
import {
  Card,
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
      <Card style={{ height: '20em', backgroundColor: '#EDEDED' }}></Card>
      <Card style={{ backgroundColor: '#151515', marginTop: '1em' }}>

        <Grid>
          <GridItem span={4}>

          </GridItem>
          <GridItem span={4} rowSpan={12}>
            <Flex>
              <FlexItem>
                <a href="https://cd.foundation">
                  <img src="https://tekton.dev/partner-logos/cdf.png"
                    alt="tekton.dev" />
                </a>
              </FlexItem>
            </Flex>
            <Flex style={{ justifyContent: 'center' }}>
              <TextContent>
                <Text component={TextVariants.h1}
                  style={{ color: 'white' }}>
                  Tekton is a{' '}
                  <Text component={TextVariants.a} href="https://cd.foundation">
                    Continuous Delivery Foundation
                   </Text>{' '}project.
                </Text>
              </TextContent>
            </Flex>
            <Flex style={{ justifyContent: 'center' }}>
              <FlexItem>
                <img src={tekton} alt="Tekton"
                  style={{ height: '6em', marginBottom: '-1em' }} />
              </FlexItem>
            </Flex>
            <Flex>
              <CardFooter>
                <Text style={{ color: 'white', textAlign: 'center' }}>
                  © 2020 The Linux Foundation®. All rights reserved.
                  The Linux Foundation has registered trademarks and
                  uses trademarks. For a list of trademarks of
                  The Linux Foundation, please see our {' '}
                  <Text component={TextVariants.a}
                    href="https://www.linuxfoundation.org/trademark-usage/">
                    Trademark Usage page
                  </Text>
                  .{' '}Linux is a registered trademark of Linus Torvalds.
                   {' '}
                  <Text component={TextVariants.a}
                    href="https://www.linuxfoundation.org/privacy/" >
                    Privacy Policy
                   </Text>
                  {' '} and {' '}
                  <Text component={TextVariants.a}
                    href="https://www.linuxfoundation.org/terms/">
                    Terms of Use
                  </Text>
                  {' '}.
                </Text>
              </CardFooter>
            </Flex>

          </GridItem>

          <GridItem span={4}>

          </GridItem>
        </Grid>
      </Card>
    </div >
  );
};

export default Footer;
