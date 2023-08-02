FROM python

COPY ./barnacle-net /bin/barnacle-net

COPY ./scripts /scripts

RUN pip3 install inky[rpi,example-depends]
RUN pip3 install Pillow

ENTRYPOINT 	["/bin/barnacle-net"]
