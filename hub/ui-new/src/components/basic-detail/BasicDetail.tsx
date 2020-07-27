import React, {useState} from 'react';
import {
  Card,
  Flex,
  FlexItem,
  Button,
  Grid,
  Badge,
  GridItem,
  CardHead,
  TextContent,
  Text,
  CardActions,
  ClipboardCopy,
  ClipboardCopyVariant,
  Modal,
  TextVariants,
  DropdownItem,
  DropdownToggle,
  Dropdown,
} from '@patternfly/react-core';
import {
  GithubIcon,
  BuildIcon,
  DomainIcon,
  CatIcon,
  CertificateIcon,
  UserIcon,
} from '@patternfly/react-icons';
import '@patternfly/react-core/dist/styles/base.css';
import './basicdetail.css';
import {fetchTaskDescription} from '../redux/Actions/TaskActionDescription';
import {connect} from 'react-redux';
import Rating from '../rating/Rating';
import {API_URL} from '../../constants';


export interface LatestVersionInfo {
  id: number,
  version: string,
  displayName: string,
  description: string,
  minPipelinesVersion: string,
  rawURL: string,
  webURL: string,
  updatedAt: string,
}
export interface CatalogInfo {
  id: number,
  type: string,
}
export interface TagInfo {
  id: number,
  name: string,
}
export interface BasicDetailPropObject {
  id: any
  name: string;
  type: string,
  catalog: CatalogInfo,
  latestVersion: LatestVersionInfo,
  tags: Array<TagInfo>,
  rating: number;
}

export interface Version {
  id: number,
  version: string,
  rawURL: string,
  webURL: string,
}

export interface BasicDetailProp {
  task: BasicDetailPropObject
  version: Array<Version>
}


const BasicDetail: React.FC<BasicDetailProp> = (props: any) => {
  React.useEffect(() => {
    props.fetchTaskDescription(props.task.latestVersion.rawURL);
    // eslint-disable-next-line
  }, [])


  const taskArr: any = [];
  const [isModalOpen, setIsModalOpen] = useState(false);

  const [summary, setSummary] = useState(
    props.task.latestVersion.description.substring(0,
      props.task.latestVersion.description.indexOf('\n')) ||
    props.task.latestVersion.description,
  );

  const [descrption, setDescription] =
    useState(
      props.task.latestVersion.description.indexOf('\n') !== -1 ?
        props.task.latestVersion.description.substring(
          props.task.latestVersion.description.indexOf('\n') +
          1).trim() : ' ');

  const [versions, setVersion] =
    useState(props.task.latestVersion.version + ' (latest) ');
  const [taskLink, setTaskLink] =
    useState(`kubectl apply -f ${ props.task.latestVersion.rawURL }`);

  const [href, setHref] = useState(`${
    props.task.latestVersion.webURL.substring(0,
      props.task.latestVersion.webURL.lastIndexOf('/') + 1) }`);

  // Display Name for resource
  let displayName = '';
  if (props.task.latestVersion.displayName === '') {
    displayName = props.task.name;
  } else {
    displayName = props.task.latestVersion.displayName.replace(
      /(^\w|\s+\w){1}/g, ((str) => {
        return str.toUpperCase();
      })) + ' (' + (props.task.name) + ')';
  }

  // Dropdown menu to show versions
  const [isOpen, set] = useState(false);
  const dropdownItems: any = [];

  if (props.version) {
    fetchTaskDescription(props.task.latestVersion.rawURL);
    const tempTaskData = props.version.reverse();
    tempTaskData.forEach((item: any, index: any) => {
      if (props.task.latestVersion.version === item.version) {
        dropdownItems.push(<DropdownItem
          key={`res-${ item.version }`} name={item.id.toString()}
          onClick={getVersionDetail} >
          {item.version + ' (latest) '}
        </DropdownItem>);
      } else {
        dropdownItems.push(<DropdownItem
          key={`res-${ item.version }`} name={item.id.toString()}
          onClick={getVersionDetail} >
          {item.version}
        </DropdownItem>);
      }
    });
  }

  // versions details of a perticular version
  function getVersionDetail(event: any) {
    fetch(`${ API_URL }/resource/version/${ event.target.name }`)
      .then((response) => response.json())
      .then((data) => {
        props.fetchTaskDescription(data.rawURL);

        setHref(`${ data.webURL.substring(0,
          data.webURL.lastIndexOf('/') + 1) }`);

        setTaskLink(`kubectl apply -f ${ data.rawURL }`);

        setSummary(data.description.substring(0,
          data.description.indexOf('\n')) ||
          props.task.latestVersion.description);

        setDescription(
          props.task.latestVersion.description.indexOf('\n') !== -1 ?
            props.task.latestVersion.description.substring(
              props.task.latestVersion.description.indexOf('\n') +
              1).trim() : ' ',
        );

        displayName = data.dsiplayName === ' ' ? data.resource.name :
          displayName.replace(/(^\w|\s+\w){1}/g, ((str) => {
            return str.toUpperCase();
          }));
      });
    setVersion(event.target.text);
  }

  const ontoggle = (isOpen: React.SetStateAction<boolean>) => set(isOpen);
  const onSelect = () => set(!isOpen);

  // Get tags for resource
  if (props.task.tags != null) {
    props.task.tags.forEach((item: any) => {
      taskArr.push(item.name);
    });
  } else {
    taskArr.push([]);
  }

  //  ading icon for details page
  let resourceIcon: React.ReactNode;
  if (props.task.type.toLowerCase() === 'task') {
    resourceIcon = <BuildIcon
      style={{height: '5em', width: '5em'}} color="#484848" />;
  } else {
    resourceIcon = <DomainIcon
      style={{height: '5em', width: '5em'}} color="#4848484" />;
  }

  // for verification status of resources
  let verifiedStatus: any;
  if (props.task) {
    if (props.task.catalog.type.toLowerCase() === 'official') {
      verifiedStatus = <div className="vtask" >
        <CatIcon size="md" color='#484848'
          style={{width: '2em', height: '1.7em'}} />
      </div>;
    }
    if (props.task.catalog.type.toLowerCase() === 'verified') {
      verifiedStatus = <div className="vtask" >
        <CertificateIcon size="md" color='#484848'
          style={{width: '2em', height: '1.7em'}} />
      </div>;
    }
    if (props.task.catalog.type.toLowerCase() === 'community') {
      verifiedStatus = <div className="vtask" >
        <UserIcon size="md" color='#484848'
          style={{width: '2em', height: '1.7em'}} />
      </div>;
    }
  }

  return (
    <Flex>

      <Card style={{
        marginLeft: '-2em', marginRight: '-2em',
        marginTop: '-2em', width: '120%', paddingBottom: '2em',
      }}>
        <CardHead style={{paddingTop: '2em'}}>
          <div style={{height: '7em', paddingLeft: '10em', marginTop: '5em'}}>
            {resourceIcon}
          </div>

          <TextContent style={{paddingLeft: '4em', paddingTop: '2em'}}>

            <Flex breakpointMods={[{modifier: 'row', breakpoint: 'lg'}]}>

              <FlexItem>
                <Text style={{fontSize: '2em'}}>
                  {/* {props.task.name.charAt(0).toUpperCase() +
                    props.task.name.slice(1)} */}
                  {displayName}
                </Text>
              </FlexItem>

              <FlexItem>
                {verifiedStatus}
              </FlexItem>

            </Flex>

            <Text style={{fontSize: '1em'}}>
              <GithubIcon size="md"
                style={{marginRight: '0.5em', marginBottom: '-0.3em'}} />

              <a href={href} target="_">Github</a>
            </Text>

            <Grid>

              <GridItem span={10}
                style={{paddingBottom: '1.5em', textAlign: 'justify'}}>

                {summary}
                <br />
                <br />
                {descrption}

              </GridItem>


              <GridItem>
                {
                  taskArr.map((tag: any) => {
                    return (
                      <Badge
                        style={{
                          paddingRight: '1em',
                          marginBottom: '1em', marginRight: '1em',
                        }}
                        key={tag}
                        className="badge">{tag}
                      </Badge>);
                  })
                }
              </GridItem>

            </Grid>

          </TextContent>

          <CardActions style={{marginRight: '3em', paddingTop: '2em'}}>

            <Flex breakpointMods={[{modifier: 'column', breakpoint: 'lg'}]}>
              <FlexItem>
                <Rating />
              </FlexItem>

              <FlexItem style={{marginLeft: '-3em'}}>
                <React.Fragment>
                  {document.queryCommandSupported('copy')}
                  <Button variant="primary"
                    className="button"
                    onClick={() => setIsModalOpen(!isModalOpen)}
                    style={{width: '8.5em'}}
                  >
                    Install
                  </Button>

                  <Modal
                    width={'60%'}
                    title={props.task.name.charAt(0).toUpperCase() +
                      props.task.name.slice(1)}
                    isOpen={isModalOpen}
                    onClose={() => setIsModalOpen(!isModalOpen)}
                    isFooterLeftAligned
                  >
                    <hr />
                    <div>

                      <TextContent>
                        <Text component={TextVariants.h2} className="modaltext">
                          Install on Kubernetes
                        </Text>
                        {/* {pipelineLink} */}
                        <Text> Tasks </Text>

                        <ClipboardCopy isReadOnly
                          variant={ClipboardCopyVariant.expansion}>{taskLink}
                        </ClipboardCopy>

                      </TextContent>

                      <br />
                    </div>

                  </Modal>

                </React.Fragment>

              </FlexItem>

              <FlexItem style={{marginLeft: '-2em', marginTop: '0.7em'}}>

                <Dropdown style={{marginLeft: '-1em'}}
                  onSelect={onSelect}
                  toggle={
                    <DropdownToggle
                      onToggle={ontoggle}
                      style={{width: '8.5em'}}>
                      {versions}
                    </DropdownToggle>}
                  isOpen={isOpen}
                  dropdownItems={dropdownItems}
                />

              </FlexItem>

            </Flex>

          </CardActions>

        </CardHead>

      </Card>

    </Flex >
  );
};

const mapStateToProps = (state: any) => {
  return {
    TaskDescription: state.TaskDescription.TaskDescription,
  };
};

export default connect(mapStateToProps,
  {fetchTaskDescription})(BasicDetail);
