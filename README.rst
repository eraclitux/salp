==============================================
SALP - Slackbot Assistant for Lazy Programmers
==============================================

It would like to help programmers automating recurrent boring stuff.

It's a rought, work in progress, alpha quality code.

Do not use in production until HMAC auth is implemented.

.. contents::

What can it do
==============

- intercept **GitHub** ``push`` webhooks and send a digest to its channels of them
- receive json messages on ``/message`` via authenticated POST and echoes them to Slack
- fetch https://istheinternetonfire.com/ when asked

Setup GitHub webhooks
=====================

- go on you repo's *GitHub* settings page
- click on ``Webhooks & services`` section
- set ``Payload URL`` as ``<yourdomain.tld>:<httpport>/gh-webhooks`` (default ``httpport`` is 8080)
