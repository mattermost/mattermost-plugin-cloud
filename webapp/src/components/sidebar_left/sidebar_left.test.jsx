import React from 'react';
import {render, fireEvent} from '@testing-library/react';

import SidebarLeft from './sidebar_left';

jest.mock('mattermost-redux/utils/theme_utils', () => ({
    makeStyleFromTheme: jest.fn((style) => style),
    // eslint-disable-next-line no-unused-vars
    changeOpacity: jest.fn((color, opacity) => color),
}));

jest.mock('react-bootstrap', () => ({
    Tooltip: jest.fn(({children}) => children),
    OverlayTrigger: jest.fn(({children}) => children),
}));

describe('SidebarLeft', () => {
    const id = 'user123';
    const installs = [1, 2, 3];
    const actions = {
        getCloudUserData: jest.fn(),
    };
    const showRHS = jest.fn();
    const theme = {
        sidebarText: '#ffffff',
    };

    afterEach(() => {
        jest.clearAllMocks();
    });

    it('should render the component', () => {
        const {getByTestId} = render(
            <SidebarLeft
                id={id}
                installs={installs}
                actions={actions}
                showRHS={showRHS}
                theme={theme}
            />,
        );

        const tooltip = getByTestId('yourCloudInstallationsTestId');
        expect(tooltip).toBeInTheDocument();
    });

    it('should call the showRHS function when the link is clicked', () => {
        const {getByTestId} = render(
            <SidebarLeft
                id={id}
                installs={installs}
                actions={actions}
                showRHS={showRHS}
                theme={theme}
            />,
        );

        fireEvent.click(getByTestId('yourCloudInstallationsTestId'));

        expect(showRHS).toHaveBeenCalledWith(true);
    });

    it('should call the getCloudUserData action creator on mount', () => {
        render(
            <SidebarLeft
                id={id}
                installs={installs}
                actions={actions}
                showRHS={showRHS}
                theme={theme}
            />,
        );

        expect(actions.getCloudUserData).toHaveBeenCalledWith(id);
    });

    it('should render the correct number of installs', () => {
        const {getByText} = render(
            <SidebarLeft
                id={id}
                installs={installs}
                actions={actions}
                showRHS={showRHS}
                theme={theme}
            />,
        );

        expect(getByText('3')).toBeInTheDocument();
    });

    it('should apply the correct style to the link', () => {
        const {getByTestId} = render(
            <SidebarLeft
                id={id}
                installs={installs}
                actions={actions}
                showRHS={showRHS}
                theme={theme}
            />,
        );

        const link = getByTestId('yourCloudInstallationsTestId');
        expect(link).toHaveStyle('color: #ffffff');
        expect(link).toHaveStyle('display: block');
        expect(link).toHaveStyle('margin-bottom: 10px');
        expect(link).toHaveStyle('width: 100%');
    });
});