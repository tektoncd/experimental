
import React from 'react';
import './index.css';
import imgAvatar from '../assets/logo/imgAvatar.png';
import {
  Link,
  BrowserRouter as Router,
  Route,
} from 'react-router-dom';
import SearchBar from '../search-bar/SearchBar';
import TaskContainer from '../task-container/TaskContainer';
// import UploadTask from '../upload-task/UploadTask';
import '@patternfly/react-core/dist/styles/base.css';
import logo from '../assets/logo/main.png';
import Filter from '../filter/Filter';
// import UserProfile from '../user-profile/UserProfile';
import {
  Button,
  ButtonVariant,
  ToolbarItem,
  Page,
  Brand,
  PageHeader,
  PageSection,
  Toolbar,
  ToolbarGroup,
  Avatar,
  Grid,
  GridItem,
} from '@patternfly/react-core';
import Detail from '../detail/Detail';
import BasicDetailParent from '../basic-detail/BasicDetailParent';
import BackgroundImageHeader from '../background-image/BackgroundImage';
import Login from '../Authentication/Login';
import Footer from '../footer/Footer';
interface mainProps {

}
interface mainState {
  value: string;
}

const App: React.FC<mainProps> = () => {
  const logoProps = {
    href: '/',
    // eslint-disable-next-line no-console
    onClick: () => console.log('clicked logo'),
    target: '',
  };
  const logoutUser = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('usetrID');
    window.location.assign('/');
  };
  let userimage: any;
  let displayUpload: any = '';
  let authenticationButton;
  if (localStorage.getItem('token') === null) {
    authenticationButton = <Link to="/login">
      <span
        style={{
          marginRight: '1em',
          color: 'white',
          fontSize: '1em',
        }}>
        Login</span>
    </Link>;
    displayUpload = '';
  } else {
    authenticationButton = <Link to="/">
      <span style={{marginRight: '1em', color: 'white', fontSize: '1em'}}
        onClick={logoutUser}> Logout </span>
    </Link>;

    // TODO -> commented upload feature

    // displayUpload = <Link to="/upload">
    //   <span >
    //     <PlusIcon size="sm" color='white' />
    //   </span>
    //   {' '}
    // </Link>;

    userimage = <Avatar
      style={{
        width: '1.5em',
        height: '1.5em',
      }}
      src={imgAvatar} alt="" />;
  }

  // code for header contents
  const PageToolbar = (
    // eslint-disable-next-line react/jsx-filename-extension
    <div>
      <Toolbar>
        <ToolbarGroup>
          <ToolbarItem style={{color: 'white'}}>
            {displayUpload}
            <Button id="default-example-uid-01"
              aria-label="Notifications actions"
              variant={ButtonVariant.plain}>
            </Button>
          </ToolbarItem>
          <ToolbarItem>
            {
              authenticationButton
            }


          </ToolbarItem>
          <ToolbarItem>
            {userimage}
          </ToolbarItem>
        </ToolbarGroup>
      </Toolbar>
    </div>

  );
  const Header = (
    <PageHeader
      logo={<Brand src={logo} alt="Tekton Hub Logo" />}
      logoProps={logoProps}
      toolbar={PageToolbar}
    />
  );

  return (
    <Router>
      <Page header={Header}>
        <Route exact path="/" component={BackgroundImageHeader} />
        <PageSection >
          <Grid gutter='sm' sm={6} md={4} lg={4} xl2={1}>
            <GridItem span={12}>
              <Route exact path="/detail/:taskId"
                component={BasicDetailParent} />
              <Route exact path="/detail/:taskId" component={Detail} />
              {/* <Route exact path="/upload" component={UploadTask} /> */}
            </GridItem>
            <GridItem span={2} rowSpan={12}>

              <Route exact path="/" component={Filter} />
            </GridItem>
            <GridItem span={8} rowSpan={12}>
              <Route exact path="/" component={SearchBar} />

              <Route exact path="/" component={TaskContainer} />

            </GridItem>
            <GridItem span={2} rowSpan={12}>

            </GridItem>

          </Grid>

        </PageSection>

        <PageSection>
          <Route path='/login' component={Login} />
          <Route path='/logout' component={Login} />
        </PageSection>

        <Footer />

      </Page>
    </Router>
  );
};

export default App;
