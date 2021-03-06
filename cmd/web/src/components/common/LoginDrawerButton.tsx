import React, { useRef } from "react";
import { defineMessages, useIntl } from "react-intl";
import {
  Drawer,
  DrawerBody,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  Button,
  useDisclosure,
} from "@chakra-ui/core";
import LoginForm from "./LoginForm";

const messages = defineMessages({
  account: {
    id: "login.drawer_button.account",
    defaultMessage: "Account",
  },
});

const LoginDrawerButton = () => {
  const intl = useIntl();
  const btnRef = useRef(null);
  const { isOpen, onOpen, onClose } = useDisclosure();

  return (
    <div>
      <Button variant="outline" width="full" ref={btnRef} onClick={onOpen}>
        {intl.formatMessage(messages.account)}
      </Button>
      <Drawer
        isOpen={isOpen}
        onClose={onClose}
        finalFocusRef={btnRef}
        size="full"
      >
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton border="none" />
          <DrawerBody display="flex">
            <LoginForm onSuccess={onClose} />
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </div>
  );
};

export default LoginDrawerButton;
