import React from 'react';
import {API_URL} from '../../constants';
import UserProfileChild from './UserProfileChild';
import {
  EmptyState,
  EmptyStateIcon,
  EmptyStateBody,
  Title,
  Button,
  EmptyStateVariant,
} from '@patternfly/react-core';
import {CubesIcon} from '@patternfly/react-icons';
import {Link} from 'react-router-dom';

const UserProfile: React.FC = (props: any) => {
  const userGithubId = localStorage['usetrID'];
  const [userResource, setUserResource] = React.useState();
  React.useEffect(() => {
    fetch(`${API_URL}/resources/user/${userGithubId}`)
      .then((response) => response.json())
      .then((data: any) => setUserResource(data));
    // eslint-disable-next-line
  }, []);


  if (userResource !== undefined) {
    if (userResource.length > 0) {
      return (
        <div>
          {
            <>
              <Title headingLevel="h5" size="lg"
                style={{marginLeft: '1.5em', fontSize: '2em'}}>
                My Resources
              </Title>
              <UserProfileChild task={userResource} />
            </>
          }

        </div>
      );
    }
    if (userResource.length === 0) {
      return (
        <EmptyState variant={EmptyStateVariant.full}
          style={{
            position: 'absolute', top: '10em',
            bottom: 0, right: 0, left: 0, margin: 'auto',
          }}>
          <EmptyStateIcon icon={CubesIcon} />
          <Title headingLevel="h1" size="lg">
            My Resources
          </Title>
          <EmptyStateBody>
            It seems you haven&apos;t uploaded any resources.
          </EmptyStateBody>
          <br />
          <Link to="/upload"><Button variant="primary">Upload</Button></Link>

        </EmptyState>
      );
    }
  };


  return (
    <div></div>
  );
};

export default UserProfile;
