/* eslint-disable max-len */
/* eslint-disable react/prop-types */
import React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import TimeAgo from 'javascript-time-ago';
import en from 'javascript-time-ago/locale/en';
import {
  Link,
} from 'react-router-dom';
import './index.css';
import {
  Card,
  Badge,
  GalleryItem,
  TextContent,
  CardHead,
  CardHeader,
  CardFooter,
  CardBody,
  CardActions,
  FlexItem,
  Flex,
  FlexModifiers,
  Tooltip,
} from '@patternfly/react-core';
import {
  StarIcon,
  BuildIcon,
  DomainIcon,
  CatIcon,
  CertificateIcon,
  UserIcon,
} from '@patternfly/react-icons';


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
export interface TaskPropObject {
  id: number,
  name: string;
  type: string;
  catalog: CatalogInfo;
  latestVersion: LatestVersionInfo,
  tags: Array<TagInfo>,
  rating: number;
}

export interface TaskProp {
  task: TaskPropObject
}

// eslint-disable-next-line
const Task: React.FC<TaskProp> = (props: any) => {
  const tempArr: any = [];
  if (props.task.tags != null) {
    props.task.tags.forEach((item: any) => {
      tempArr.push(item.name);
    });
  } else {
    tempArr.push([]);
  }

  TimeAgo.addLocale(en);

  // Create relative date/time formatter.
  const timeAgo = new TimeAgo('en-US');

  const catalogDate = new Date(props.task.latestVersion.updatedAt);

  const diffDays = timeAgo.format(catalogDate.getTime() - 60 * 1000);


  // for verification status of resources
  let verifiedStatus: any;
  if (props.task) {
    if (props.task.catalog.type.toLowerCase() === 'official') {
      verifiedStatus = <Tooltip content={<b>Official</b>}>
        <div className="vtask" >
          <CatIcon size="md" color='#484848'
            style={{width: '2em', height: '2em'}} />
        </div>
      </Tooltip>;
    }
    if (props.task.catalog.type.toLowerCase() === 'verified') {
      verifiedStatus = <Tooltip content={<b>Verified</b>}>
        <div className="vtask" >
          <CertificateIcon size="md" color='#484848'
            style={{width: '2em', height: '2em'}} />
        </div>
      </Tooltip>;
    }
    if (props.task.catalog.type.toLowerCase() === 'community') {
      verifiedStatus = <Tooltip content={<b>Community</b>}>
        <div className="vtask" >
          <UserIcon size="md" color='#484848'
            style={{width: '2em', height: '2em'}} />
        </div>
      </Tooltip>;
    }
  }

  // }
  // for adding icon to task and pipeline
  let resourceIcon: React.ReactNode;
  if (props.task.type.toLowerCase() === 'task') {
    resourceIcon = <Tooltip content={<b>Task</b>}>
      <BuildIcon
        style={{width: '2em', height: '2em', verticalAlign: '-0.2em'}}
        color="#484848"
      />
    </Tooltip>;
  } else {
    resourceIcon = <Tooltip content={<b>Pipeline</b>}>
      <DomainIcon
        style={{width: '2em', height: '2em', verticalAlign: '-0.2em'}}
        color="#484848"
      />
    </Tooltip>;
  };


  // Display name

  const resourceName = props.task.latestVersion.displayName === '' ?
    <span style={{fontFamily: 'courier, monospace'}}>
      {props.task.name}</span> : <span>
      {props.task.latestVersion.displayName}</span>;

  return (
    <GalleryItem>
      <Link to={'/detail/' + props.task.id}>
        <Card className="card" isHoverable
          style={{marginBottom: '1em', borderRadius: '0.5em'}}>

          <CardHead>

            <Flex breakpointMods={[{modifier: 'row', breakpoint: 'lg'}]}>

              <FlexItem style={{marginRight: '1em'}}>
                {resourceIcon}
              </FlexItem>

              <FlexItem>
                {verifiedStatus}
              </FlexItem>

            </Flex>

            <CardActions className="cardActions">

              <StarIcon style={{color: '#484848', height: '1.7em', width: '1.7em'}} />
              <TextContent className="text">
                {props.task.rating.toFixed(1)}
              </TextContent>

            </CardActions>
          </CardHead>
          <CardHeader className="catalog-tile-pf-header">

            <Flex>
              <FlexItem>
                <span className="task-heading">
                  {resourceName}
                  {/* {props.task.name[0].toUpperCase() + props.task.name.slice(1)} */}
                </span>
              </FlexItem>
              <FlexItem
                breakpointMods={[{modifier: FlexModifiers['align-right']}]}
                style={{marginBottom: '0.5em'}}>
                <span>
                  v{props.task.latestVersion.version}
                </span>
              </FlexItem>
            </Flex>
          </CardHeader>
          <CardBody className="catalog-tile-pf-body">
            <div className="catalog-tile-pf-description">
              <span>
                {`${props.task.latestVersion.description.substring(0,
                  props.task.latestVersion.description.indexOf('\n'))}` ||
                  `${props.task.latestVersion.description}`}
              </span>
            </div>

          </CardBody>
          <CardFooter className="catalog-tile-pf-footer">


            <TextContent className="text"
              style={{
                marginBottom: '0.5em', marginTop:
                  '-1em', marginLeft: '0em',
              }}>
              Updated {diffDays}
            </TextContent>

            <div style={{height: '2em'}}>{
              tempArr.slice(0, 3).map((tag: any, index: number) => (
                <Badge
                  style={{marginRight: '0.3em', marginBottom: '0.5em'}}
                  key={`badge-${tag}`}
                  className="badge">{tag}</Badge>
              ))
            }
            </div>

          </CardFooter>
        </Card>
      </Link>
    </GalleryItem >
  );
};
export default Task;
