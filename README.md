# go-htmlemailer 
a simple utility to send html based emails via SMTP or Mailgun

the overall goal is to offer a way to send the verification step of your signups or to send an email for password resets.

The SSL/TLS part was adapted from  https://gist.github.com/chrisgillis/10888032

This is a very simple email module you can use to send html based emails.  

I include an example email template that is used with the test.

Please note, in order to use Mailgun, you will need the keys you were issued by Mailgun and 

You will need to verify your domain before you can send test emails to their system through

the api.  If you do try to use the api with just a sandbox testing domain they issue, you will get an error like

Rejected: '....' Sandbox subdomains are for test purposes only. Please add your own domain or add the address to authorized recipients in Account Settings.: 

